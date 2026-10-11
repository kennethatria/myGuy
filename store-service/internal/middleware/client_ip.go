package middleware

import "github.com/gin-gonic/gin"

// ClientIPHeader carries the visitor's IP: nginx sets it from the
// NodeBalancer's PROXY header (templates/nginx-https.conf.j2).  Trusting it
// is safe because only nginx can reach this service (it listens on
// 127.0.0.1).  Gin's default reads X-Forwarded-For first, which nginx
// passes on from the client, so anyone could choose their IP.  The same as
// the backend's api.ClientIPHeader.
const ClientIPHeader = "X-Real-IP"

// TrustNginx makes c.ClientIP() (request log, traces) the visitor's IP from
// nginx, never a header the client sent.
func TrustNginx(r *gin.Engine) {
	r.TrustedPlatform = ClientIPHeader
}
