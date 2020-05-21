package formulas

import (
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"ritchie-server/server"
	"ritchie-server/server/tm"
)

type Handler struct {
	Config        server.Config
	Authorization server.Constraints
}


const (
	repoNameHeader = "x-repo-name"
	authorizationHeader = "Authorization"
)

func NewConfigHandler(config server.Config, auth server.Constraints) server.DefaultHandler {
	return Handler{Config: config, Authorization: auth}
}

func (lh Handler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var method = r.Method
		if http.MethodGet == method {
			lh.processGet(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

func (lh Handler) processGet(w http.ResponseWriter, r *http.Request) {
	org := r.Header.Get(server.OrganizationHeader)
	repos, err := lh.Config.ReadRepositoryConfig(org)
	if err != nil {
		log.Printf("Error while processing %v's repository configuration: %v", org, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if repos == nil {
		log.Println("No repository configDummy found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	repoName := r.Header.Get(repoNameHeader)
	repo, err := tm.FindRepo(repos, repoName)
	if err != nil {
		log.Printf("no repo for org %s, with name %s, error: %v", org, repoName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	bt := r.Header.Get(authorizationHeader)
	allow, err := tm.FormulaAllow(lh.Authorization, r.URL.Path, bt, org, repo)
	if err != nil {
		log.Printf("error try allow access: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !allow {
		log.Printf("Not allow access path: %s", r.URL.Path)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	bucket := "ritchie-test-bucket234376412767550"
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("sa-east-1")},
	)
	if err != nil {
		log.Printf("Failed to create session aws to path: %s, error: %v", r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	buf := &aws.WriteAtBuffer{}
	downloader := s3manager.NewDownloader(sess)
	_, err = downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(r.URL.Path),
		})
	if err != nil {
		log.Printf("Failed to read bucket: %s, error: %v", r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(buf.Bytes())
}