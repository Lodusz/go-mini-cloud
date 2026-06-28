package main

import (
	"context"
	"go-mini-cloud/internal/handler"
	"go-mini-cloud/internal/storage"
	"go-mini-cloud/internal/worker"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const storageDir = "./cloud_storage_data"

func main() {
	engine, err := storage.NewFileEngine(storageDir)
	if err != nil {
		log.Fatalf("Failed to init storage engine FATAL ERR: %v", err)
	}

	fileHandler := handler.NewFileHandler(engine)

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", fileHandler.Upload)
	mux.HandleFunc("/download/", fileHandler.Download)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	ctx, cancel := context.WithCancel(context.Background())

	gc := worker.NewGCWorker(engine, storageDir, 1*time.Minute)
	go gc.Start(ctx)

	go func() {
		log.Println("Mini-Cloud Engine запущен на :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Server crash: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Получен сигнал завершения. Остановка сервера...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Принудительное завершение сервера: %v", err)
	}

	log.Println("Сервер успешно остановлен")
}
