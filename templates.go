package readas

import (
	"html/template"
	"io"
	"log"
)

var templates = map[string]*template.Template{}

const templatesDir = "templates/"

func init() {
	initTemplate("post")
	initTemplate("index")
}

func initTemplate(name string) {
	templates[name] = template.Must(template.New(name).ParseFiles(templatesDir+name+".tmpl", templatesDir+"base.tmpl"))
}

// renderTemplate retrieves the given template and renders it to the given io.Writer.
// If something goes wrong, the error is logged and returned.
func renderTemplate(w io.Writer, tmpl string, data interface{}) error {
	err := templates[tmpl].ExecuteTemplate(w, tmpl, data)
	if err != nil {
		log.Printf("[ERROR] Error rendering %s: %s\n", tmpl, err)
	}

	return err
}
