package main

import (
	cfg "github.com/conductorone/baton-asana/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("asana", cfg.Config)
}
