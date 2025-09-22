package service

import (
	"url-shortner-be/model/transaction"
	"url-shortner-be/model/user"
	"url-shortner-be/module/repository"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type TransactionService struct {
	db           *gorm.DB
	repository   repository.Repository
	associations []string
}

func NewTransactionService(db *gorm.DB, repo repository.Repository) *TransactionService {
	return &TransactionService{
		db:           db,
		repository:   repo,
		associations: []string{},
	}
}

func (service *TransactionService) CreateTransaction(uow *repository.UnitOfWork, userId uuid.UUID, amount float32, transactionType, note string) error {

	user := &user.User{}
	err := service.repository.GetRecord(uow, user, repository.Filter("id = ?", userId))
	if err != nil {
		return err
	}

	transaction := &transaction.Transaction{
		Amount: amount,
		Type:   transactionType,
		Note:   note,
		UserID: user.ID,
	}
	transaction.CreatedBy = userId

	err = service.repository.Add(uow, transaction)
	if err != nil {
		return err
	}

	return nil
}
