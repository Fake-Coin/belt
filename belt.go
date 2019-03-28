package belt

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"goji.io/pat"
	"vallon.me/shortening"
)

type BeltCtx struct {
	*Belt
	db *gorm.DB
}

func (ctx *BeltCtx) Save() error {
	if res := ctx.db.Save(&ctx.Belt); res.Error != nil {
		return res.Error
	}
	return nil
}

func (ctx *BeltCtx) SaveWithOptions(opts []Option) error {
	for _, o := range opts {
		if o.ID != 0 {
			for _, beltopt := range ctx.Belt.Options {
				if beltopt.ID == o.ID {
					if res := ctx.db.Model(&beltopt).Update(&o); res.Error != nil {
						return res.Error
					}
					log.Println(beltopt)
					break
				}
			}
		} else {
			if res := ctx.db.Model(&ctx.Belt).Association("Options").Append(o); res.Error != nil {
				return res.Error
			}
		}
	}

	return ctx.db.Save(&ctx.Belt).Error
}

func Context(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		beltID, err := shortening.Decode([]byte(pat.Param(r, "beltid")))
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		db, ok := r.Context().Value("database").(*gorm.DB)
		if !ok {
			log.Println("could not find database in context")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var b Belt
		if res := db.Set("gorm:auto_preload", true).First(&b, beltID); res.Error != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), "beltctx", &BeltCtx{&b, db})

		h.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func Edit(w http.ResponseWriter, r *http.Request) {
	belt, ok := r.Context().Value("beltctx").(*BeltCtx)
	if !ok {
		log.Println("could not find belt in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := editTmpl.ExecuteTemplate(w, "edit.html", belt); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func SaveEdit(w http.ResponseWriter, r *http.Request) {
	belt, ok := r.Context().Value("beltctx").(*BeltCtx)
	if !ok {
		log.Println("could not find belt in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := r.ParseMultipartForm(0); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339Nano, r.FormValue("endTime"))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	belt.Title = r.FormValue("title")
	belt.Message = strings.Replace(r.FormValue("description"), "\r\n", "\n", -1)
	belt.EndTime = endTime

	opts, err := getOptions(r.MultipartForm.Value)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := belt.SaveWithOptions(opts); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"ok": true}`)
}

func getOptions(form map[string][]string) ([]Option, error) {
	ids, ok1 := form["option[id][]"]
	names, ok2 := form["option[name][]"]
	imgs, ok3 := form["option[img][]"]

	if (!ok1 || !ok2 || !ok3) || len(ids) != len(names) || len(ids) != len(imgs) {
		return nil, errors.New("bad option data")
	}

	opts := make([]Option, len(ids))
	for i, id := range ids {
		n, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		if 0 < n {
			opts[i].ID = uint(n)
		}
	}

	for i, name := range names {
		opts[i].Name = name
	}

	for i, img := range imgs {
		opts[i].Image = img
	}

	return opts, nil
}
