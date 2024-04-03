package service

import (
	"github.com/mateusmacedo/go-nunu-template/internal/repository"
	"github.com/mateusmacedo/go-nunu-template/pkg/jwt"
	"github.com/mateusmacedo/go-nunu-template/pkg/log"
	"github.com/mateusmacedo/go-nunu-template/pkg/sid"
)

type Service struct {
	logger *log.Logger
	sid    *sid.Sid
	jwt    *jwt.JWT
	tm     repository.Transaction
}

func NewService(tm repository.Transaction, logger *log.Logger, sid *sid.Sid, jwt *jwt.JWT) *Service {
	return &Service{
		logger: logger,
		sid:    sid,
		jwt:    jwt,
		tm:     tm,
	}
}
