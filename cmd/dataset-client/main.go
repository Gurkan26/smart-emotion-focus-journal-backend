package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/gurkanfikretgunak/masterfabric-go/internal/infrastructure/dataset"
)

const (
	defaultDatasetRepo = "Gurkan26/enterprise-prompt-optimizer"
	defaultPort        = "8081"
)

func main() {
	hfToken := os.Getenv("HF_TOKEN")
	datasetRepo := os.Getenv("HF_DATASET_REPO")
	if datasetRepo == "" {
		datasetRepo = defaultDatasetRepo
	}

	client := dataset.NewClient(hfToken)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Println("==========================================================================")
	fmt.Println("🚀 MasterFabric Academy - Hugging Face Datasets Client (Go 1.24+ / Chi)")
	fmt.Printf("[*] Target Dataset: %s\n", datasetRepo)
	fmt.Println("==========================================================================")

	// 1. CLI Demonstration: Fetch and render dataset rows immediately
	data, err := client.FetchFirstRows(ctx, datasetRepo)
	if err != nil {
		fmt.Printf("[!] Notice: Unable to fetch live dataset rows directly (%v)\n", err)
	} else {
		fmt.Printf("[+] Successfully fetched %d dataset rows from Hugging Face Datasets Server:\n\n", len(data.Rows))
		for _, item := range data.Rows {
			fmt.Printf("   [%d] Template: %-10s | Original: %-25s -> Optimized: %s\n",
				item.RowIdx, item.Row.Template, item.Row.OriginalPrompt, item.Row.OptimizedPrompt)
		}
	}
	fmt.Println("==========================================================================")

	// 2. HTTP REST API Server using Chi Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "dataset": datasetRepo})
	})

	r.Get("/api/v1/dataset/prompts", func(w http.ResponseWriter, r *http.Request) {
		reqCtx, reqCancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer reqCancel()

		respData, err := client.FetchFirstRows(reqCtx, datasetRepo)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(respData)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	fmt.Printf("\n[*] Starting REST API Server on http://localhost:%s\n", port)
	fmt.Printf("[*] Dataset endpoint: http://localhost:%s/api/v1/dataset/prompts\n", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
