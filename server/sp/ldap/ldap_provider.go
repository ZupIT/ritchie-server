package ldap

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/jtblin/go-ldap-client"

	"ritchie-server/server"
)

const (
	base               = "base"
	host               = "host"
	serverName         = "serverName"
	port               = "port"
	useSSL             = "useSSL"
	skipTLS            = "skipTLS"
	insecureSkipVerify = "insecureSkipVerify"
	bindDN             = "bindDN"
	bindPassword       = "bindPassword"
	userFilter         = "userFilter"
	groupFilter        = "groupFilter"
	attributeUsername  = "attributeUsername"
	attributeName      = "attributeName"
	attributeEmail     = "attributeEmail"
	ttl                = "ttl"
)

type ldapError struct {
	code int
	err  error
}

type ldapUser struct {
	roles    []string
	userInfo server.UserInfo
}

type lConfig struct {
	base               string
	host               string
	serverName         string
	port               int
	useSSL             bool
	skipTLS            bool
	insecureSkipVerify bool
	bindDN             string
	bindPassword       string
	userFilter         string
	groupFilter        string
	attributeUsername  string
	attributeName      string
	attributeEmail     string
	ttl                int64
}

type ldapConfig struct {
	client *ldap.LDAPClient
	config lConfig
}

func NewLdapProvider(config map[string]string) server.SecurityManager {
	cf := loadLConfig(config)
	cl := loadClient(cf)
	return ldapConfig{
		client: cl,
		config: cf,
	}
}

func loadClient(cf lConfig) *ldap.LDAPClient {
	att := []string{cf.attributeName, cf.attributeUsername, cf.attributeUsername}
	return &ldap.LDAPClient{
		Base:         cf.base,
		Host:         cf.host,
		ServerName:   cf.serverName,
		InsecureSkipVerify: cf.insecureSkipVerify,
		Port:         cf.port,
		UseSSL:       cf.useSSL,
		SkipTLS:      cf.skipTLS,
		BindDN:       cf.bindDN,
		BindPassword: cf.bindPassword,
		UserFilter:   cf.userFilter,
		GroupFilter:  cf.groupFilter,
		Attributes:   att,
	}
}

func loadLConfig(config map[string]string) lConfig {
	p, _ := strconv.Atoi(config[port])
	us, _ := strconv.ParseBool(config[useSSL])
	st, _ := strconv.ParseBool(config[skipTLS])
	isv, _ := strconv.ParseBool(config[insecureSkipVerify])
	ttl, _ := strconv.ParseInt(config[ttl], 10, 64)
	return lConfig{
		base:               config[base],
		host:               config[host],
		serverName:         config[serverName],
		port:               p,
		useSSL:             us,
		skipTLS:            st,
		insecureSkipVerify: isv,
		bindDN:             config[bindDN],
		bindPassword:       config[bindPassword],
		userFilter:         config[userFilter],
		groupFilter:        config[groupFilter],
		attributeUsername:  config[attributeUsername],
		attributeName:      config[attributeName],
		attributeEmail:     config[attributeEmail],
		ttl:                ttl,
	}
}

func (k ldapConfig) TTL() int64 {
	return k.config.ttl
}

func (k ldapConfig) Login(username, password string) (server.User, server.LoginError) {
	defer k.client.Close()
	ok, user, err := k.client.Authenticate(username, password)
	if err != nil {
		return nil, ldapError {
			code: 401,
			err:  err,
		}
	}
	if !ok {
		return nil, ldapError {
			code: 401,
			err:  errors.New(fmt.Sprintf("Authenticating failed for user %s", username)),
		}
	}
	groups, err := k.client.GetGroupsOfUser(username)
	if err != nil {
		return nil, ldapError {
			code: 500,
			err:  errors.New(fmt.Sprintf("Error getting groups for user %s", username)),
		}
	}
	lu := ldapUser {
		roles: groups,
		userInfo: server.UserInfo{
			Name:     user[attributeName],
			Username: username,
			Email:    user[attributeEmail],
		},
	}
	return lu, nil
}

/*func main() {
	client := &ldap.LDAPClient{
		Base:         "dc=example,dc=org",
		Host:         "localhost",
		ServerName:   "ldap.example.org",
		InsecureSkipVerify: false,
		Port:         389,
		UseSSL:       false,
		SkipTLS:      true,
		BindDN:       "cn=admin,dc=example,dc=org",
		BindPassword: "admin",
		UserFilter:   "(uid=%s)",
		GroupFilter:  "(memberUid=%s)",
		Attributes:   []string{"givenName", "sn", "mail", "uid"},
	}
	// It is the responsibility of the caller to close the connection
	defer client.Close()

	ok, user, err := client.Authenticate("user", "user")
	if err != nil {
		log.Fatalf("Error authenticating user %s: %+v", "user", err)
	}
	if !ok {
		log.Fatalf("Authenticating failed for user %s", "user")
	}
	log.Printf("User: %+v", user)

	groups, err := client.GetGroupsOfUser("user")
	if err != nil {
		log.Fatalf("Error getting groups for user %s: %+v", "user", err)
	}
	log.Printf("Groups: %+v", groups)
}*/

func (le ldapError) Error() error {
	return le.err
}
func (le ldapError) Code() int {
	return le.code
}

func (u ldapUser) Roles() []string {
	return u.roles
}
func (u ldapUser) UserInfo() server.UserInfo {
	return u.userInfo
}