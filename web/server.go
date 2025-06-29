package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router *gin.Engine
	port   string
}

func NewServer(port string) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	server := &Server{
		router: router,
		port:   port,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	s.router.GET("/healthz", s.healthCheck)
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/", s.healthCheck)
	s.router.GET("/metrics", s.metrics)
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Bot is up",
		"service": "meme-bot",
	})
}

func (s *Server) metrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"metrics": gin.H{
			"uptime": "running",
			// todo: количество обработанных сообщений, ошибки, время отклика OpenAI и т.д.
		},
	})
}

func (s *Server) Start() error {
	return s.router.Run(":" + s.port)
}

func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
