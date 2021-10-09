// photoweb project main.go
package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strings"
)

const (
	ListDir      = 0x0001
	UPLOAD_DIR   = "uploads"    // 存放上传文件的路径
	TEMPLATE_DIR = "./www/html" // 存放静态页面文件的路径
)

var templates = make(map[string]*template.Template) //用于存放缓存的模板

func init() {
	fileInfoArr, err := ioutil.ReadDir(TEMPLATE_DIR)
	if err != nil {
		panic(err)
	}

	var templateName, templatePath string
	for _, fileInfo := range fileInfoArr {
		templateName = fileInfo.Name()
		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}

		templatePath = TEMPLATE_DIR + "/" + templateName
		realName := realName(templateName)
		t := template.Must(template.ParseFiles(templatePath))
		templates[realName] = t
	}
}
func realName(tmpl string) string {
	name := path.Base(tmpl)
	if "." == name || " " == name {
		return name
	}

	var nameArr []string
	nameArr = strings.Split(name, ".html")
	return nameArr[0]
}

func main() {
	mux := http.NewServeMux()
	staticDirHandler(mux, "/assets/", "./public", 0)
	http.HandleFunc("/", safeHandler(listHandler))
	http.HandleFunc("/upload", safeHandler(uploadHandler))
	http.HandleFunc("/view", safeHandler(viewHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				log.Printf("WARN:panic in %v - %v \n", fn, e)
				log.Println(string(debug.Stack()))
			}
		}()
		fn(w, r)
	}
}

func staticDirHandler(mux *http.ServeMux, prefix string, staticDir string, flags int) {
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		file := staticDir + r.URL.Path[len(prefix)-1:]
		if (flags & ListDir) == 0 {
			if exists := isExists(file); !exists {
				http.NotFound(w, r)
				return
			}
		}
		http.ServeFile(w, r, file)
	})
}

func readerHtml(w http.ResponseWriter, tmpl string, locals map[string]interface{}) (err error) {
	err = templates[tmpl].Execute(w, locals)
	return err
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if "GET" == r.Method {
		err := readerHtml(w, "upload", nil)
		check(err)
	} else if "POST" == r.Method {
		f, h, err := r.FormFile("image")
		check(err)
		filename := h.Filename
		defer f.Close()
		t, err := os.Create(UPLOAD_DIR + "/" + filename)
		check(err)
		defer t.Close()
		_, err = io.Copy(t, f)
		check(err)
		http.Redirect(w, r, "/view?id="+filename, http.StatusFound)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId
	if exists := isExists(imagePath); !exists {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/"+path.Ext(imagePath)[1:])
	http.ServeFile(w, r, imagePath)
}
func isExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	fileInfoArr, err := ioutil.ReadDir(UPLOAD_DIR)
	check(err)

	locals := make(map[string]interface{})
	images := []string{}
	for _, fileInfo := range fileInfoArr {
		images = append(images, fileInfo.Name())
	}
	locals["images"] = images
	err = readerHtml(w, "list", locals)
	check(err)
}

//模板渲染,recover，匿名函数,defer,静态模板
