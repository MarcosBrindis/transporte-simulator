package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScenarioInfo contiene información de un escenario disponible
type ScenarioInfo struct {
	ID       string // "parada_normal", "mi_escenario_custom"
	Name     string // "Parada Normal", "Mi Escenario Custom"
	Source   string // "builtin" o "yaml"
	FilePath string // Ruta al archivo YAML (si es yaml)
}

// DiscoverScenarios encuentra todos los escenarios disponibles
func DiscoverScenarios(yamlDir string) []ScenarioInfo {
	scenarios := make([]ScenarioInfo, 0)

	// 1. Agregar escenarios predefinidos (builtin)
	scenarios = append(scenarios, ScenarioInfo{
		ID:     "parada_normal",
		Name:   "Parada Normal",
		Source: "builtin",
	})

	scenarios = append(scenarios, ScenarioInfo{
		ID:     "parada_con_salidas",
		Name:   "Parada con Salidas",
		Source: "builtin",
	})

	scenarios = append(scenarios, ScenarioInfo{
		ID:     "circuito_completo",
		Name:   "Circuito Completo",
		Source: "builtin",
	})

	// 2. Buscar archivos YAML en el directorio
	if yamlDir != "" {
		yamlScenarios := discoverYAMLScenarios(yamlDir)
		scenarios = append(scenarios, yamlScenarios...)
	}

	return scenarios
}

// discoverYAMLScenarios busca archivos .yaml/.yml en un directorio
func discoverYAMLScenarios(dir string) []ScenarioInfo {
	scenarios := make([]ScenarioInfo, 0)

	// Verificar si el directorio existe
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return scenarios
	}

	// Leer archivos del directorio
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("⚠️  Error leyendo directorio de escenarios: %v\n", err)
		return scenarios
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Solo archivos .yaml o .yml
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		// Crear ID y Name a partir del nombre de archivo
		fileName := file.Name()
		baseName := strings.TrimSuffix(fileName, ext)

		// ID: nombre del archivo sin extensión
		id := "yaml_" + baseName

		// Name: capitalizar y reemplazar _ por espacios
		name := strings.ReplaceAll(baseName, "_", " ")
		name = strings.Title(name) // Capitalizar

		filePath := filepath.Join(dir, fileName)

		scenarios = append(scenarios, ScenarioInfo{
			ID:       id,
			Name:     name + " (YAML)",
			Source:   "yaml",
			FilePath: filePath,
		})

		fmt.Printf("📄 Escenario YAML detectado: %s (%s)\n", name, filePath)
	}

	return scenarios
}
