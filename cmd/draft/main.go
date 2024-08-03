package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type JobType int

const (
	HTTPRequestJob JobType = iota
	DBUpdateJob
	ProcessJob
)

type Job struct {
	Type     JobType
	RecordID string
	URL      string
	Data     string
}

type SQSMessage struct {
	RecordID string
	URL      string
}

func initializeDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./jobs.db")
	if err != nil {
		log.Fatalf("Erro ao abrir o banco de dados: %v", err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		record_id TEXT,
		url TEXT,
		status TEXT,
		data TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Erro ao criar tabela: %v", err)
	}

	return db
}

func saveJob(db *sql.DB, message SQSMessage, status string, data string) error {
	insertJobSQL := `INSERT INTO jobs (record_id, url, status, data) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(insertJobSQL, message.RecordID, message.URL, status, data)
	time.Sleep(2 * time.Second) // Simular atraso
	return err
}

func updateJobStatus(db *sql.DB, recordID string, status string, data string) error {
	updateJobSQL := `UPDATE jobs SET status = ?, data = ? WHERE record_id = ?`
	_, err := db.Exec(updateJobSQL, status, data, recordID)
	time.Sleep(2 * time.Second) // Simular atraso
	return err
}

func getUnprocessedJobs(db *sql.DB) ([]SQSMessage, error) {
	query := `SELECT record_id, url FROM jobs WHERE status = 'error' OR status = 'timeout'`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []SQSMessage
	for rows.Next() {
		var message SQSMessage
		err := rows.Scan(&message.RecordID, &message.URL)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// Simulação de requisição HTTP
func makeHTTPRequest(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	time.Sleep(2 * time.Second) // Simular atraso
	return string(body), nil
}

// Worker que processa requisições HTTP
func httpWorker(id int, db *sql.DB, jobs <-chan Job, results chan<- Job, errors chan<- SQSMessage, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d encerrando devido ao contexto finalizado (tempo limite atingido)\n", id)
			errors <- SQSMessage{RecordID: job.RecordID, URL: job.URL}
			updateJobStatus(db, job.RecordID, "timeout", "")
			return
		default:
			if job.Type != HTTPRequestJob {
				continue
			}
			log.Printf("Worker %d iniciando requisição HTTP para %s\n", id, job.URL)
			result, err := makeHTTPRequest(ctx, job.URL)
			if err != nil {
				errors <- SQSMessage{RecordID: job.RecordID, URL: job.URL}
				log.Printf("Worker %d encontrou um erro ao processar a requisição HTTP para %s: %v", id, job.URL, err)
				updateJobStatus(db, job.RecordID, "error", "")
				continue
			}
			job.Data = result
			job.Type = DBUpdateJob
			results <- job
			updateJobStatus(db, job.RecordID, "http_done", result)
			log.Printf("Worker %d concluiu requisição HTTP para %s\n", id, job.URL)
		}
	}
}

// Atualização real no banco de dados SQLite
func updateDatabase(db *sql.DB, recordID string, data string) error {
	// Atualizar o status no banco de dados
	err := updateJobStatus(db, recordID, "db_updated", data)
	if err != nil {
		return fmt.Errorf("erro ao atualizar banco de dados para recordID %s: %v", recordID, err)
	}

	log.Printf("Database updated: %s -> %s\n", recordID, data)
	return nil
}

// Worker que processa atualizações no banco de dados
func dbWorker(id int, db *sql.DB, jobs <-chan Job, results chan<- Job, errors chan<- SQSMessage, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d encerrando devido ao contexto finalizado (tempo limite atingido)\n", id)
			errors <- SQSMessage{RecordID: job.RecordID, URL: job.URL}
			updateJobStatus(db, job.RecordID, "timeout", job.Data)
			return
		default:
			if job.Type != DBUpdateJob {
				continue
			}
			log.Printf("Worker %d iniciando atualização do banco de dados para %s\n", id, job.RecordID)
			err := updateDatabase(db, job.RecordID, job.Data)
			if err != nil {
				errors <- SQSMessage{RecordID: job.RecordID, URL: job.URL}
				log.Printf("Worker %d encontrou um erro ao atualizar o banco de dados para %s: %v", id, job.RecordID, err)
				updateJobStatus(db, job.RecordID, "error", job.Data)
				continue
			}
			job.Type = ProcessJob
			results <- job
			log.Printf("Worker %d concluiu atualização do banco de dados para %s\n", id, job.RecordID)
		}
	}
}

// Função que simula o processamento de um job e pode causar erro
func processJob(recordID string) error {
	if time.Now().UnixNano()%2 == 0 {
		return fmt.Errorf("erro simulado no processamento do job: %s", recordID)
	}
	fmt.Printf("Job processado com sucesso: %s\n", recordID)
	time.Sleep(2 * time.Second) // Simular atraso
	return nil
}

// Worker que processa jobs gerais
func processWorker(id int, db *sql.DB, jobs <-chan Job, errors chan<- SQSMessage, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d encerrando devido ao contexto finalizado (tempo limite atingido)\n", id)
			errors <- SQSMessage{RecordID: job.RecordID, URL: job.URL}
			updateJobStatus(db, job.RecordID, "timeout", job.Data)
			return
		default:
			if job.Type != ProcessJob {
				continue
			}
			log.Printf("Worker %d iniciando processamento do job: %s\n", id, job.RecordID)
			err := processJob(job.RecordID)
			if err != nil {
				errors <- SQSMessage{RecordID: job.RecordID, URL: job.URL}
				log.Printf("Worker %d encontrou um erro ao processar o job: %s. Erro: %v", id, job.RecordID, err)
				updateJobStatus(db, job.RecordID, "error", job.Data)
				continue
			}
			updateJobStatus(db, job.RecordID, "processed", job.Data)
			log.Printf("Worker %d concluiu processamento do job: %s\n", id, job.RecordID)
		}
	}
}

func lambdaHandler(ctx context.Context, db *sql.DB, messages []SQSMessage) ([]SQSMessage, []SQSMessage) {
	workerCount := getWorkerCount()

	httpJobs := make(chan Job)
	dbJobs := make(chan Job)
	processJobs := make(chan Job)
	errors := make(chan SQSMessage, len(messages))
	var wg sync.WaitGroup

	// Iniciando workers para requisições HTTP
	for w := 1; w <= workerCount; w++ {
		wg.Add(1)
		go httpWorker(w, db, httpJobs, dbJobs, errors, ctx, &wg)
	}

	// Iniciando workers para atualizações no banco de dados
	for w := 1; w <= workerCount; w++ {
		wg.Add(1)
		go dbWorker(w, db, dbJobs, processJobs, errors, ctx, &wg)
	}

	// Iniciando workers para processamento de jobs gerais
	for w := 1; w <= workerCount; w++ {
		wg.Add(1)
		go processWorker(w, db, processJobs, errors, ctx, &wg)
	}

	// Enviando jobs de requisição HTTP
	for _, message := range messages {
		httpJobs <- Job{Type: HTTPRequestJob, RecordID: message.RecordID, URL: message.URL}
	}

	close(httpJobs)
	wg.Wait()
	close(dbJobs)
	close(processJobs)
	close(errors)

	// Coletar registros não processados e erros
	var unprocessedMessages []SQSMessage
	for errMsg := range errors {
		unprocessedMessages = append(unprocessedMessages, errMsg)
	}

	return unprocessedMessages, unprocessedMessages
}

func getWorkerCount() int {
	workerCountStr := os.Getenv("WORKER_COUNT")
	if workerCountStr == "" {
		return 3 // Valor padrão
	}
	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		log.Printf("Valor inválido para WORKER_COUNT: %s. Usando valor padrão de 3.\n", workerCountStr)
		return 3
	}
	return workerCount
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	db := initializeDB()
	defer db.Close()

	initialMessages := []SQSMessage{
		{RecordID: "1", URL: "https://jsonplaceholder.typicode.com/todos/1"},
		{RecordID: "2", URL: "https://jsonplaceholder.typicode.com/todos/2"},
		{RecordID: "3", URL: "https://jsonplaceholder.typicode.com/todos/3"},
		{RecordID: "4", URL: "https://jsonplaceholder.typicode.com/todos/4"},
		{RecordID: "5", URL: "https://jsonplaceholder.typicode.com/todos/5"},
		{RecordID: "6", URL: "https://jsonplaceholder.typicode.com/todos/6"},
		{RecordID: "7", URL: "https://jsonplaceholder.typicode.com/todos/7"},
		{RecordID: "8", URL: "https://jsonplaceholder.typicode.com/todos/8"},
		{RecordID: "9", URL: "https://jsonplaceholder.typicode.com/todos/9"},
		{RecordID: "10", URL: "https://jsonplaceholder.typicode.com/todos/10"},
	}

	// Salvar jobs iniciais no banco de dados
	for _, msg := range initialMessages {
		saveJob(db, msg, "received", "")
	}

	attempt := 1
	globalTimeout := 180 * time.Second
	processingComplete := false

	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	for len(initialMessages) > 0 && !processingComplete {
		log.Printf("Ciclo de processamento #%d\n", attempt)

		// Passa as mensagens para a função de processamento
		unprocessedMessages, errorMessages := lambdaHandler(ctx, db, initialMessages)

		if len(errorMessages) > 0 {
			log.Printf("%d registros não foram processados corretamente. Detalhes: %v\n", len(errorMessages), errorMessages)
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Deseja reprocessar os registros com erro? (sim/não): ")
			answer, _ := reader.ReadString('\n')
			if answer == "sim\n" {
				initialMessages = append(initialMessages, errorMessages...)
			} else {
				processingComplete = true
			}
		}

		if len(unprocessedMessages) > 0 {
			log.Printf("%d registros não foram processados e serão reprocessados\n", len(unprocessedMessages))
			initialMessages = unprocessedMessages
		} else {
			log.Println("Todos os registros foram processados com sucesso")
			processingComplete = true
		}

		attempt++
	}

	if ctx.Err() == context.DeadlineExceeded {
		log.Println("Timeout global atingido. Encerrando o programa.")
	}
}
