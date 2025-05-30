package onyx

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type CSRFProtection struct {
	secret       []byte
	tokenLength  int
	cookieName   string
	headerName   string
	formField    string
	sessionKey   string
	secure       bool
	httpOnly     bool
	sameSite     http.SameSite
	maxAge       time.Duration
	exemptPaths  []string
	trustOrigins []string
}

func NewCSRFProtection() *CSRFProtection {
	secret := make([]byte, 32)
	rand.Read(secret)
	
	return &CSRFProtection{
		secret:      secret,
		tokenLength: 32,
		cookieName:  "csrf_token",
		headerName:  "X-CSRF-TOKEN",
		formField:   "_token",
		sessionKey:  "csrf_token",
		secure:      false,
		httpOnly:    true,
		sameSite:    http.SameSiteLaxMode,
		maxAge:      24 * time.Hour,
		exemptPaths: []string{},
		trustOrigins: []string{},
	}
}

func (csrf *CSRFProtection) WithSecret(secret []byte) *CSRFProtection {
	csrf.secret = secret
	return csrf
}

func (csrf *CSRFProtection) WithTokenLength(length int) *CSRFProtection {
	csrf.tokenLength = length
	return csrf
}

func (csrf *CSRFProtection) WithCookieName(name string) *CSRFProtection {
	csrf.cookieName = name
	return csrf
}

func (csrf *CSRFProtection) WithHeaderName(name string) *CSRFProtection {
	csrf.headerName = name
	return csrf
}

func (csrf *CSRFProtection) WithFormField(field string) *CSRFProtection {
	csrf.formField = field
	return csrf
}

func (csrf *CSRFProtection) WithSecure(secure bool) *CSRFProtection {
	csrf.secure = secure
	return csrf
}

func (csrf *CSRFProtection) WithMaxAge(maxAge time.Duration) *CSRFProtection {
	csrf.maxAge = maxAge
	return csrf
}

func (csrf *CSRFProtection) WithExemptPaths(paths []string) *CSRFProtection {
	csrf.exemptPaths = paths
	return csrf
}

func (csrf *CSRFProtection) WithTrustOrigins(origins []string) *CSRFProtection {
	csrf.trustOrigins = origins
	return csrf
}

func (csrf *CSRFProtection) GenerateToken() string {
	token := make([]byte, csrf.tokenLength)
	rand.Read(token)
	return hex.EncodeToString(token)
}

func (csrf *CSRFProtection) VerifyToken(token, sessionToken string) bool {
	if token == "" || sessionToken == "" {
		return false
	}
	
	return subtle.ConstantTimeCompare([]byte(token), []byte(sessionToken)) == 1
}

func (csrf *CSRFProtection) isExemptPath(path string) bool {
	for _, exemptPath := range csrf.exemptPaths {
		if strings.HasPrefix(path, exemptPath) {
			return true
		}
	}
	return false
}

func (csrf *CSRFProtection) isTrustedOrigin(origin string) bool {
	for _, trustedOrigin := range csrf.trustOrigins {
		if origin == trustedOrigin {
			return true
		}
	}
	return false
}

func (csrf *CSRFProtection) getTokenFromRequest(c Context) string {
	// Try to get token from header
	if token := c.Header(csrf.headerName); token != "" {
		return token
	}
	
	// TODO: Implement form parsing with interface-based system
	// For now, only support header-based tokens
	// Full form parsing would require extending the Context interface
	
	return ""
}

func (csrf *CSRFProtection) getSessionToken(c Context) string {
	// TODO: Implement session support with interface-based system
	// For now, use context storage as a temporary workaround
	if token, exists := c.Get(csrf.sessionKey); exists {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	return ""
}

func (csrf *CSRFProtection) setSessionToken(c Context, token string) {
	// TODO: Implement session support with interface-based system
	// For now, use context storage as a temporary workaround
	c.Set(csrf.sessionKey, token)
}

func (csrf *CSRFProtection) setCookieToken(c Context, token string) {
	// TODO: Implement cookie support with interface-based system
	// For now, just set as header
	c.SetHeader("Set-Cookie", fmt.Sprintf("%s=%s; Path=/", csrf.cookieName, token))
	
	/*
	cookie := &http.Cookie{
		Name:     csrf.cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(csrf.maxAge.Seconds()),
		Secure:   csrf.secure,
		HttpOnly: csrf.httpOnly,
		SameSite: csrf.sameSite,
	}
	
	c.SetCookie(cookie)
	*/
}

func CSRFMiddleware(options ...*CSRFProtection) MiddlewareFunc {
	csrf := NewCSRFProtection()
	if len(options) > 0 {
		csrf = options[0]
	}
	
	return func(c Context) error {
		path := c.URL()
		method := c.Method()
		
		if csrf.isExemptPath(path) {
			return c.Next()
		}
		
		origin := c.GetHeader("Origin")
		if origin != "" && !csrf.isTrustedOrigin(origin) {
			return c.JSON(403, map[string]string{
				"error": "Origin not trusted",
			})
		}
		
		sessionToken := csrf.getSessionToken(c)
		if sessionToken == "" {
			sessionToken = csrf.GenerateToken()
			csrf.setSessionToken(c, sessionToken)
			csrf.setCookieToken(c, sessionToken)
		}
		
		c.Set("csrf_token", sessionToken)
		
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return c.Next()
		}
		
		requestToken := csrf.getTokenFromRequest(c)
		if !csrf.VerifyToken(requestToken, sessionToken) {
			return c.JSON(419, map[string]string{
				"error": "CSRF token mismatch",
			})
		}
		
		return c.Next()
	}
}

func GetCSRFToken(c Context) string {
	if token, exists := c.Get("csrf_token"); exists {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	
	// TODO: Add session support when available
	return ""
}

func GetCSRFField(c Context) string {
	token := GetCSRFToken(c)
	return fmt.Sprintf(`<input type="hidden" name="_token" value="%s">`, token)
}

type CSRFTokenMismatchException struct {
	message string
}

func (e *CSRFTokenMismatchException) Error() string {
	return e.message
}

func NewCSRFTokenMismatchException() *CSRFTokenMismatchException {
	return &CSRFTokenMismatchException{
		message: "CSRF token mismatch",
	}
}

func VerifyCSRFToken(c Context) error {
	csrf := NewCSRFProtection()
	
	sessionToken := csrf.getSessionToken(c)
	requestToken := csrf.getTokenFromRequest(c)
	
	if !csrf.VerifyToken(requestToken, sessionToken) {
		return NewCSRFTokenMismatchException()
	}
	
	return nil
}

type CSRFHelper struct {
	csrf *CSRFProtection
}

func NewCSRFHelper() *CSRFHelper {
	return &CSRFHelper{
		csrf: NewCSRFProtection(),
	}
}

func (ch *CSRFHelper) Token(c Context) string {
	return GetCSRFToken(c)
}

func (ch *CSRFHelper) Field(c Context) string {
	return GetCSRFField(c)
}

func (ch *CSRFHelper) Meta(c Context) string {
	token := GetCSRFToken(c)
	return fmt.Sprintf(`<meta name="csrf-token" content="%s">`, token)
}

func (ch *CSRFHelper) Header(c Context) string {
	return ch.csrf.headerName
}

func DoubleSubmitCookieMiddleware() MiddlewareFunc {
	return func(c Context) error {
		method := c.Method()
		
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			token := generateCSRFToken()
			
			cookie := &http.Cookie{
				Name:     "csrf_token",
				Value:    token,
				Path:     "/",
				MaxAge:   86400,
				Secure:   false,
				HttpOnly: false,
				SameSite: http.SameSiteLaxMode,
			}
			
			c.SetCookie(cookie)
			c.Set("csrf_token", token)
			
			return c.Next()
		}
		
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil {
			return c.JSON(419, map[string]string{
				"error": "CSRF token not found in cookie",
			})
		}
		
		headerToken := c.GetHeader("X-CSRF-TOKEN")
		if headerToken == "" {
			headerToken = c.PostForm("_token")
		}
		
		if headerToken == "" {
			return c.JSON(419, map[string]string{
				"error": "CSRF token not found in request",
			})
		}
		
		if subtle.ConstantTimeCompare([]byte(cookieToken.Value), []byte(headerToken)) != 1 {
			return c.JSON(419, map[string]string{
				"error": "CSRF token mismatch",
			})
		}
		
		return c.Next()
	}
}

func generateCSRFToken() string {
	token := make([]byte, 32)
	rand.Read(token)
	return hex.EncodeToString(token)
}

func SameSiteCSRFMiddleware() MiddlewareFunc {
	return func(c Context) error {
		method := c.Method()
		
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return c.Next()
		}
		
		origin := c.Header("Origin")
		referer := c.Header("Referer")
		host := c.Header("Host")
		
		if origin != "" {
			if !isValidOrigin(origin, host) {
				return c.JSON(403, map[string]string{
					"error": "Invalid origin",
				})
			}
		} else if referer != "" {
			if !isValidReferer(referer, host) {
				return c.JSON(403, map[string]string{
					"error": "Invalid referer",
				})
			}
		} else {
			return c.JSON(403, map[string]string{
				"error": "Missing origin or referer header",
			})
		}
		
		return c.Next()
	}
}

func isValidOrigin(origin, host string) bool {
	return strings.Contains(origin, host)
}

func isValidReferer(referer, host string) bool {
	return strings.Contains(referer, host)
}