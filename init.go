package belt

import (
	"fmt"

	"html/template"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/jinzhu/gorm"
)

var editTmpl *template.Template
var beltTmpl *template.Template

func init() {
	fmt.Println(AssetNames())

	editTmpl = template.Must(template.New("edit.html").Parse(string(MustAsset("assets/edit.html"))))

	beltTmpl = template.Must(template.New("index.html").Parse(string(MustAsset("assets/index.html"))))
	beltTmpl = template.Must(beltTmpl.New("payModal").Parse(string(MustAsset("assets/payModal.html"))))
	beltTmpl = template.Must(beltTmpl.New("footer").Parse(string(MustAsset("assets/footer.html"))))
	beltTmpl = template.Must(beltTmpl.New("belt").Parse(string(MustAsset("assets/belt.html"))))
}

func AssetFS() *assetfs.AssetFS {
	return assetFS()
}

func Migrate(db *gorm.DB) error {
	if res := db.AutoMigrate(&Belt{}, &Option{}, &Bet{}, &BetTx{}); res.Error != nil {
		return res.Error
	}
	return nil
}
