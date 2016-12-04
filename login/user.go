// package login is used for handling user credentials.
// By default, we want to use digest authentication and bcrypt! "github.com/Compufreak345/go-http-auth"  "golang.org/x/crypto/bcrypt"
// Important : As this uses gorilla sessions, wrap your handlers with a GorillaClearHandler
package login

import (
	"crypto/rand"
	"crypto/rsa"
	"sync"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/tools"
)

// IUser defines the minimum user object.
type IUser interface {
	LoginName() string
	SetLoginName(string)
	FirstName() string
	SetFirstName(string)
	LastName() string
	SetLastName(string)
	Title() string
	SetTitle(string)
	Email() string
	SetEmail(string)
	Id() int64
	SetId(int64)
	Ip() string
	SetIp(string)
	PwHash() string
	SetPwHash(string)
	Nonce() *PrivKeyAndSalt

	// IsLoggedIn() returns true if this User-object is logged in.
	IsLoggedIn() bool
	OpenSessionIds() map[string]bool

	AddOpenSessionId(string)
	RemoveOpenSessionId(string)
}

const uTag = dbg.Tag("webfw/login/user.go")

var OpenSessionIdMapsByUserId map[int64]map[string]bool

// User defines the default user object
type User struct {
	IloginName         string
	IfirstName         string
	IlastName          string
	Ititle             string
	Iemail             string
	Iid                int64
	Iip                string
	lastNonce          *PrivKeyAndSalt
	nonceWasUsed       bool
	notFirstNonce      bool
	IisLoggedIn        bool
	IopenSessionIds    map[string]bool
	openSessionIdMutex sync.Mutex
	ipwHash            string
}

// init initializes the User-Session-Map - it is currently not used, we use a Redistore in goodl/utils/userManager/userManager.go for this.
func init() {
	OpenSessionIdMapsByUserId = make(map[int64]map[string]bool)
}

// AddOpenSessionId is currently not used, we use a Redistore in goodl/utils/userManager/userManager.go for this.
func (u *User) AddOpenSessionId(sessionId string) {
	u.openSessionIdMutex.Lock()
	defer u.openSessionIdMutex.Unlock()
	if u.IopenSessionIds == nil {
		InitOpenSessionIds(u.IopenSessionIds, u.Id())
	}
	u.IopenSessionIds[sessionId] = true
}

// InitOpenSessionIds is currently not used, we use a Redistore in goodl/utils/userManager/userManager.go for this.
func InitOpenSessionIds(osids map[string]bool, usrId int64) {
	osids = OpenSessionIdMapsByUserId[usrId]
	if osids == nil {
		osids = make(map[string]bool)
	}
	if usrId != 0 { // Save map if we are not basic empty user with ID 0
		OpenSessionIdMapsByUserId[usrId] = osids
	}
}

// RemoveOpenSessionId is currently not used, we use a Redistore in goodl/utils/userManager/userManager.go for this.
func (u *User) RemoveOpenSessionId(sessionId string) {
	u.openSessionIdMutex.Lock()
	defer u.openSessionIdMutex.Unlock()
	if u.IopenSessionIds == nil {
		InitOpenSessionIds(u.IopenSessionIds, u.Id())
	}
	delete(u.IopenSessionIds, sessionId)
}

// // OpenSessionIds is currently not used, we use a Redistore in goodl/utils/userManager/userManager.go for this.
func (u *User) OpenSessionIds() map[string]bool {
	u.openSessionIdMutex.Lock()
	defer u.openSessionIdMutex.Unlock()
	if u.IopenSessionIds == nil {
		InitOpenSessionIds(u.IopenSessionIds, u.Id())
	}
	return u.IopenSessionIds
}

// PrivKeyAndSalt ist used to store a private key and a salt
type PrivKeyAndSalt struct {
	PrivKey *rsa.PrivateKey
	Salt    []byte
}

// NewUser initializes an empty user.
func NewUser() *User {
	u := User{}

	u.IopenSessionIds = make(map[string]bool)
	return &u
}

// LoginName returns the users LoginName
func (u *User) LoginName() string {
	return u.IloginName
}
// SetLoginName sets the users LoginName
func (u *User) SetLoginName(loginName string) {
	u.IloginName = loginName
}
// FirstName returns the users first name
func (u *User) FirstName() string {
	return u.IfirstName
}
// SetFirstName sets the users first name
func (u *User) SetFirstName(firstName string) {
	u.IfirstName = firstName
}
// LastName returns the users last name
func (u *User) LastName() string {
	return u.IlastName
}
// SetLastName sets the users last name
func (u *User) SetLastName(lastName string) {
	u.IlastName = lastName
}
// Title returns the users title
func (u *User) Title() string {
	return u.Ititle
}
// SetTitle sets the users title
func (u *User) SetTitle(title string) {
	u.Ititle = title
}
// Email returns the users Email-address
func (u *User) Email() string {
	return u.Iemail
}
// SetEmail sets the users Email-address
func (u *User) SetEmail(email string) {
	u.Iemail = email
}
// Id returns the users Id
func (u *User) Id() int64 {
	return u.Iid
}
// SetId sets the users Id
func (u *User) SetId(id int64) {
	u.Iid = id
}
// Ip returns the users Ip
func (u *User) Ip() string {
	return u.Iip
}
// SetIp sets the users Ip
func (u *User) SetIp(ip string) {
	u.Iip = ip
}

// PwHash returns the users hashed password.
func (u *User) PwHash() string {
	return u.ipwHash
}
// SetPwHash sets the users hashed password
func (u *User) SetPwHash(pwHash string) {
	u.ipwHash = pwHash
}

// Nonce generates a new Nonce for logging in. (Please STILL USE SSL)
func (u *User) Nonce() *PrivKeyAndSalt {
	if !u.notFirstNonce {
		u.notFirstNonce = true
		u.nonceWasUsed = true
	}
	if u.nonceWasUsed {
		u.nonceWasUsed = false
		var err error
		u.lastNonce = &PrivKeyAndSalt{
			Salt: tools.GenerateSalt(),
		}
		u.lastNonce.PrivKey, err = rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			dbg.E(uTag, "Error generating nonce.", err)
			panic(err)
		}
		if err = u.lastNonce.PrivKey.Validate(); err != nil {
			dbg.E(uTag, "Error validating nonce.", err)
			panic(err)
		}

	} else {
		u.nonceWasUsed = true
	}
	return u.lastNonce
}

// ReqNonce represents a nonce used for login encryption (Please STILL USE SSL)
type ReqNonce struct {
	Nonce  string
	ReqId  string
	NonceN string
	NonceE int
	Salt   []byte
}

// IsLoggedIn returns true if the user is logged in.
func (u *User) IsLoggedIn() bool {
	return u.IisLoggedIn
}
