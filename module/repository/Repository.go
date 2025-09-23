package repository

import (
	"url-shortner-be/components/errors"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type Repository interface {
	Add(uow *UnitOfWork, out interface{}) error
	GetAll(uow *UnitOfWork, out interface{}, queryProcessor ...QueryProcessor) error
	GetRecord(uow *UnitOfWork, out interface{}, queryProcessors ...QueryProcessor) error
	GetCount(uow *UnitOfWork, out, count interface{}, queryProcessors ...QueryProcessor) error
	GetRecordByID(uow *UnitOfWork, tenantID uuid.UUID, out interface{}, queryProcessors ...QueryProcessor) error
	// Save(uow *UnitOfWork, value interface{}) error
	Update(uow *UnitOfWork, out interface{}, queryProcessors ...QueryProcessor) error
	UpdateWithMap(uow *UnitOfWork, model interface{}, value map[string]interface{}, queryProcessors ...QueryProcessor) error
}

type GormRepository struct{}

func NewGormRepository() *GormRepository {
	return &GormRepository{}
}

type UnitOfWork struct {
	DB        *gorm.DB
	Committed bool
	Readonly  bool
}

func NewUnitOfWork(db *gorm.DB, readonly bool) *UnitOfWork {
	commit := false
	if readonly {
		return &UnitOfWork{
			DB:        db.New(),
			Committed: commit,
			Readonly:  readonly,
		}
	}

	return &UnitOfWork{
		DB:        db.New().Begin(),
		Committed: commit,
		Readonly:  readonly,
	}
}

func (uow *UnitOfWork) RollBack() {

	if !uow.Committed && !uow.Readonly {
		uow.DB.Rollback()
	}
}

func (uow *UnitOfWork) Commit() {
	if !uow.Readonly && !uow.Committed {
		uow.Committed = true
		uow.DB.Commit()
	}
}

func executeQueryProcessors(db *gorm.DB, out interface{}, queryProcessors ...QueryProcessor) (*gorm.DB, error) {
	var err error
	for _, query := range queryProcessors {
		if query != nil {
			db, err = query(db, out)
			if err != nil {
				return db, err
			}
		}
	}
	return db, nil
}

func (repository *GormRepository) Add(uow *UnitOfWork, out interface{}) error {
	return uow.DB.Create(out).Error
}

func (repository *GormRepository) GetAll(uow *UnitOfWork, out interface{}, queryProcessors ...QueryProcessor) error {
	db := uow.DB
	// db := uow.DB.Unscoped()
	db, err := executeQueryProcessors(db, out, queryProcessors...)
	if err != nil {
		return err
	}
	return db.Debug().Find(out).Error
}

func (repository *GormRepository) GetCount(uow *UnitOfWork, out, count interface{}, queryProcessors ...QueryProcessor) error {
	db := uow.DB
	db, err := executeQueryProcessors(db, out, queryProcessors...)
	if err != nil {
		return err
	}
	return db.Debug().Model(out).Count(count).Error
}

func (repository *GormRepository) GetRecord(uow *UnitOfWork, out interface{}, queryProcessors ...QueryProcessor) error {
	db := uow.DB
	db, err := executeQueryProcessors(db, out, queryProcessors...)
	if err != nil {
		return err
	}
	return db.Debug().First(out).Error
}

func (repository *GormRepository) GetRecordByID(uow *UnitOfWork, tenantID uuid.UUID, out interface{}, queryProcessors ...QueryProcessor) error {
	// #tenantID should be the first element in slice if "where" is appeneded in QP.
	queryProcessors = append([]QueryProcessor{Filter("id = ?", tenantID)}, queryProcessors...)
	return repository.GetRecord(uow, out, queryProcessors...)
}

func Select(query interface{}, args ...interface{}) QueryProcessor {
	return func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
		db = db.Select(query, args...)
		return db, nil
	}
}

func (repository *GormRepository) UpdateWithMap(uow *UnitOfWork, model interface{}, value map[string]interface{},
	queryProcessors ...QueryProcessor) error {
	db := uow.DB
	db, err := executeQueryProcessors(db, value, queryProcessors...)
	if err != nil {
		return err
	}
	return db.Debug().Model(model).Update(value).Error
}

func PreloadAssociations(preloadAssociations []string) QueryProcessor {
	return func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
		for _, association := range preloadAssociations {
			db = db.Debug().Preload(association)
		}
		return db, nil
	}
}

func Paginate(limit, offset int, totalCount *int) QueryProcessor {
	return func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
		if out != nil {
			if totalCount != nil {
				if err := db.Model(out).Count(totalCount).Error; err != nil {
					return db, err
				}
			}
		}

		if limit != -1 {
			db = db.Limit(limit)
		}

		if offset > 0 {
			db = db.Offset(limit * offset)
		}
		return db, nil
	}
}

func Filter(condition string, args ...interface{}) QueryProcessor {
	return func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
		db = db.Debug().Where(condition, args...)
		return db, nil
	}
}

func Order(column string) QueryProcessor {
	return func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
		db = db.Order(column) //db.Order("created_at desc").First(out).Error
		return db, nil
	}
}

// CombineQueries will process slice of queryprocessors and return single queryprocessor.
func CombineQueries(queryProcessors []QueryProcessor) QueryProcessor {
	return func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
		tempDB, err := executeQueryProcessors(db, out, queryProcessors...)
		return tempDB, err
	}
}

func DoesEmailExist(db *gorm.DB, email string, out interface{}, queryProcessors ...QueryProcessor) (bool, error) {
	if email == "" {
		return false, errors.NewNotFoundError("email not present")
	}
	count := 0
	// Below comment would make the tenant check before all query processor (Uncomment only if needed in future)
	// queryProcessors = append([]QueryProcessor{Filter("tenant_id = ?", tenantID)},queryProcessors... )
	db, err := executeQueryProcessors(db, out, queryProcessors...)
	if err != nil {
		return false, err
	}
	if err := db.Debug().Model(out).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (repository *GormRepository) Update(uow *UnitOfWork, out interface{}, queryProcessors ...QueryProcessor) error {
	db := uow.DB
	db, err := executeQueryProcessors(db, out, queryProcessors...)
	if err != nil {
		return err
	}
	return db.Model(out).Update(out).Error
}
