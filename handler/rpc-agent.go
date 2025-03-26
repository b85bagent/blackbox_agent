package handler

import (
	"blackbox_agent/server"
)

type Server struct {
	ServerStruct *server.Server
}

func New(s *server.Server) *Server {
	return &Server{
		ServerStruct: s,
	}
}
