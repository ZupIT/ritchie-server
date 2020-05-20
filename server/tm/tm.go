package tm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"ritchie-server/server"
)

func TreeRemoteAllow(sec server.Constraints, bToken, org, tPath string, repo server.Repository) (server.Tree, error) {
	rTree, err := treeRemote(tPath, repo)
	if err != nil {
		return rTree, err
	}

	roles, err := sec.ListRealmRoles(bToken, org)
	if err != nil {
		return rTree, err
	}
	rfind := make(map[string]interface{})
	for _, r := range roles {
		rfind[strings.ToUpper(r.(string))] = r
	}
	ft := server.Tree{}
	ft.Version = rTree.Version
	for _, c := range rTree.Commands {
		if len(c.Roles) > 0 {
			for _, r := range c.Roles {
				if rfind[strings.ToUpper(r)] != nil {
					ft.Commands = append(ft.Commands, c)
				}
			}
		} else {
			ft.Commands = append(ft.Commands, c)
		}
	}
	if repo.ReplaceRepoUrl != "" {
		for _, c := range ft.Commands {
			if c.Formula != nil {
				if c.Formula.RepoUrl != "" {
					c.Formula.RepoUrl = repo.ReplaceRepoUrl
				}
			}
		}
	}
	return ft, nil
}

func FormulaAllow(sec server.Constraints, fPath, token, org string, repo server.Repository) (bool, error) {
	tr, err := TreeRemoteAllow(sec, token, org, repo.TreePath, repo)
	if err != nil {
		return false, err
	}
	roles, err := sec.ListRealmRoles(token, org)
	if err != nil {
		return false, err
	}

	rfind := make(map[string]interface{})
	for _, r := range roles {
		rfind[strings.ToUpper(r.(string))] = r
	}
	p := strings.Replace(fPath, "/formulas/", "", 1)
	s := strings.Split(p, "/")
	key := strings.ReplaceAll(p, "/" + s[len(s) -1], "")
	for _, c := range tr.Commands {
		if c.Formula != nil {
			if c.Formula.Path == key {
				return true, nil
			}
		}
	}
	return false, nil
}

func FindRepo(repos []server.Repository, repoName string) (server.Repository, error) {
	var repository server.Repository
	for _, r := range repos {
		if r.Name == repoName {
			repository = r
			break
		}
	}
	if repository.Name == "" {
		return repository, fmt.Errorf("No repo with name %s\n", repoName)
	}
	return repository, nil
}

func treeRemote(tPath string, repo server.Repository) (server.Tree, error) {
	var tree server.Tree
	//tURL := fmt.Sprintf("%s%s", repo.Remote, tPath)
	t, err := loadTreeFileAws(tPath)
	if err != nil {
		return tree, err
	}
	return t, nil
}

/*func loadTreeFile(url string) (server.Tree, error) {
	var response server.Tree
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return response, err
	}

	hc := &http.Client{Timeout: 5 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return response, err
	}

	if resp.StatusCode != 200 {
		return response, fmt.Errorf("%d - failed to get index for %s\n", resp.StatusCode, url)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return response, err
	}
	return response, nil
}*/

func loadTreeFileAws(treePath string) (server.Tree, error) {
	var response server.Tree
	bucket := "ritchie-test-bucket234376412767550"
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("sa-east-1")},
	)
	if err != nil {
		return response, err
	}
	buf := &aws.WriteAtBuffer{}
	downloader := s3manager.NewDownloader(sess)
	_, err = downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(treePath),
		})
	if err != nil {
		return response, err
	}

	if err := json.Unmarshal(buf.Bytes(), &response); err != nil {
		return response, err
	}
	return response, nil
}
