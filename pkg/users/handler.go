package users

import (
	"encoding/json"
	"github.com/prometheus/common/log"
	"net/http"
)

type User struct {
	Id int `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	Website string `json:"website"`
}

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get("http://jsonplaceholder.typicode.com/users")
		if err != nil {
			log.Error(err)
			w.WriteHeader(500)
			return
		}
		defer resp.Body.Close()

		var u []User
		if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
			log.Error(err.Error())
			w.WriteHeader(500)
			return
		}

		b, err := json.Marshal(u)
		if err != nil {
			log.Error(err)
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
		w.Write([]byte(b))
	}
}