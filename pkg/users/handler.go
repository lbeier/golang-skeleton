package users

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/common/log"
)

type Handler struct {
	Repository Repository
}

func NewHandler(ur Repository) Handler {
	return Handler{
		Repository: ur,
	}
}

func (h *Handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get("http://jsonplaceholder.typicode.com/users")
		if err != nil {
			log.Error(err)
			w.WriteHeader(500)
			return
		}
		defer resp.Body.Close()

		var us []User
		if err := json.NewDecoder(resp.Body).Decode(&us); err != nil {
			log.Error(err.Error())
			w.WriteHeader(500)
			return
		}

		b, err := json.Marshal(us)
		if err != nil {
			log.Error(err)
			w.WriteHeader(500)
			return
		}

		for _, u := range us {
			h.Repository.Save(u)
		}

		w.WriteHeader(200)
		w.Write([]byte(b))
	}
}
