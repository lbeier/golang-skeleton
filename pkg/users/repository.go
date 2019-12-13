package users

import (
	"database/sql"
	"log"
)

type User struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Website string `json:"website"`
}

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{
		DB: db,
	}
}

func (r *Repository) Save(u User) {
	sqlStatement := `INSERT INTO users (name, email, phone, website) VALUES ($1, $2, $3, $4)`
	_, err := r.DB.Exec(sqlStatement, u.Name, u.Email, u.Phone, u.Website)
	if err != nil {
		log.Print("Error while saving: %s", err.Error())
	}
}
