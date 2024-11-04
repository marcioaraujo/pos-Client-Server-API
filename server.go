package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type USDBRL struct {
	Code       string `json:"code"`
	CodeIn     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

func main() {
	// Inicializa o banco de dados SQLite
	db, err := sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		fmt.Println("Erro ao conectar ao banco de dados:", err)
		return
	}
	defer db.Close()

	// Cria a tabela se não existir
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS cotacoes (id INTEGER PRIMARY KEY, bid TEXT, timestamp TEXT)")
	if err != nil {
		fmt.Println("Erro ao criar tabela:", err)
		return
	}

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
		defer cancel()

		cotacao, err := BuscaCambio(ctx)
		if err != nil {
			http.Error(w, "Erro ao buscar cotação", http.StatusInternalServerError)
			fmt.Println("Erro:", err)
			return
		}

		// Salva a cotação no banco de dados com timeout de 10ms
		ctx, cancel = context.WithTimeout(r.Context(), 10*time.Millisecond)
		defer cancel()
		err = SalvaCotacao(ctx, db, cotacao.Bid)
		if err != nil {
			http.Error(w, "Erro ao salvar cotação no banco de dados", http.StatusInternalServerError)
			fmt.Println("Erro ao salvar cotação:", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"bid": cotacao.Bid})
	})

	fmt.Println("Servidor rodando na porta 8080")
	http.ListenAndServe(":8080", nil)
}

// BuscaCambio faz a requisição para a API de câmbio com contexto e timeout
func BuscaCambio(ctx context.Context) (*USDBRL, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var c struct {
		USDBRL USDBRL `json:"USDBRL"`
	}
	err = json.Unmarshal(body, &c)
	if err != nil {
		return nil, err
	}

	return &c.USDBRL, nil
}

// SalvaCotacao salva a cotação no banco de dados SQLite com contexto e timeout
func SalvaCotacao(ctx context.Context, db *sql.DB, bid string) error {
	_, err := db.ExecContext(ctx, "INSERT INTO cotacoes (bid, timestamp) VALUES (?, ?)", bid, time.Now().Format(time.RFC3339))
	return err
}
