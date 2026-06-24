package router

import (
	"net/http"

	"github.com/developer-overheid-nl/don-register-common/problem"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type CORSOptions struct {
	AllowHeaders  []string
	ExposeHeaders []string
}

func NewEngine(apiVersion string, opts CORSOptions) *gin.Engine {
	g := gin.Default()
	g.HandleMethodNotAllowed = true

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = opts.AllowHeaders
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "API-Version"}
	}
	config.ExposeHeaders = opts.ExposeHeaders
	if len(config.ExposeHeaders) == 0 {
		config.ExposeHeaders = []string{"API-Version"}
	}
	g.Use(cors.New(config))
	g.Use(APIVersionMiddleware(apiVersion))
	return g
}

func InstallProblemHandlers(g *gin.Engine, apiVersion string) {
	g.NoMethod(func(c *gin.Context) {
		apiErr := problem.New(http.StatusMethodNotAllowed, "Method not allowed")
		c.Abort()
		c.Header("API-Version", apiVersion)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(apiErr.Status, apiErr)
	})
	g.NoRoute(func(c *gin.Context) {
		apiErr := problem.NewNotFound("Resource does not exist")
		c.Abort()
		c.Header("API-Version", apiVersion)
		c.Header("Content-Type", "application/problem+json")
		c.JSON(apiErr.Status, apiErr)
	})
}

type apiVersionWriter struct {
	gin.ResponseWriter
	version string
}

func (w *apiVersionWriter) WriteHeader(code int) {
	w.Header().Set("API-Version", w.version)
	w.ResponseWriter.WriteHeader(code)
}

func APIVersionMiddleware(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer = &apiVersionWriter{ResponseWriter: c.Writer, version: version}
		c.Next()
	}
}
