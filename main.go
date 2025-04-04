package main

import (
    "database/sql"
    "encoding/json"
	"fmt"
    "log"
    "net/http"
    "os"
	"strconv"
    "strings"
    "time"
    "github.com/joho/godotenv"

    _ "github.com/lib/pq"
)

// Define la estructura de la respuesta de JSON
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Estructura para un partido 
type Match struct {
	ID int `json:"id"`
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`
	MatchDate string `json:"matchDate"`
	Goals int `json:"goals"`
    YellowCards int `json:"yellowCards"`
    RedCards int `json:"redCards"`
    ExtraTime int `json:"extraTime"`
}

// Configuracion de MiddleWare
func enableCors(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}


// Handler para validar funcionamineto de la API
func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metodo incorrecto utilizado", http.StatusMethodNotAllowed)
		return
	}
	resp := Response{
		Status:  "success",
		Message: "Pong",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Handler para jalar todas las matches
func getMatchesHandler(w http.ResponseWriter, r *http.Request) {
    enableCors(w)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "Metodo incorrecto utilizado", http.StatusMethodNotAllowed)
        return
    }

    connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        log.Println(err)
        return
    }
    defer db.Close()

    rows, err := db.Query(`
        SELECT id, home_team, away_team, 
               match_date, goals, 
               yellow_cards, red_cards, 
               extra_time
        FROM matches
    `)
    if err != nil {
        http.Error(w, "Error al obtener los partidos", http.StatusInternalServerError)
        log.Println(err)
        return
    }
    defer rows.Close()

    var matches []Match
    for rows.Next() {
        var matchDate time.Time
        var m Match
        err := rows.Scan(
            &m.ID,
            &m.HomeTeam,
            &m.AwayTeam,
            &matchDate,
            &m.Goals,
            &m.YellowCards,
            &m.RedCards,
            &m.ExtraTime,
        )
        if err != nil {
            http.Error(w, "Error al leer los datos", http.StatusInternalServerError)
            log.Println(err)
            return
        }
        m.MatchDate = matchDate.Format("2006-01-02")
        matches = append(matches, m)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(matches)
}

// Handler para hacer POST de un partido
func postMatchHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(w)

	if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }

	if r.Method != http.MethodPost {
		http.Error(w, "Metodo incorrecto utilizado", http.StatusMethodNotAllowed)
		return
	}

	var m Match
    if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
        http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
        log.Println("Error decoding JSON:", err)
        return
    }

	connStr := fmt.Sprintf(
        "host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        log.Println("Error conectando a la DB:", err)
        return
    }
    defer db.Close()

	query := `
        INSERT INTO matches (home_team, away_team, match_date)
        VALUES ($1, $2, $3)
        RETURNING id
    `
    var newID int
    err = db.QueryRow(query, m.HomeTeam, m.AwayTeam, m.MatchDate).Scan(&newID)
    if err != nil {
        http.Error(w, "Error al insertar el partido", http.StatusInternalServerError)
        log.Println("Error insertando partido:", err)
        return
    }
    m.ID = newID


	// Aplicar formato json
	w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated) // 201 Created
    if err := json.NewEncoder(w).Encode(m); err != nil {
        http.Error(w, "Error al codificar respuesta", http.StatusInternalServerError)
        log.Println("Error al codificar respuesta:", err)
        return
    }
}

// Handler para jalar un partido por su id
func getMatchByIDHandler(w http.ResponseWriter, r *http.Request) {
    enableCors(w)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "Método incorrecto utilizado", http.StatusMethodNotAllowed)
        return
    }

    idStr := strings.TrimPrefix(r.URL.Path, "/api/matches/")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "ID de partido inválido", http.StatusBadRequest)
        return
    }

    connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        log.Println(err)
        return
    }
    defer db.Close()

    var matchDate time.Time
    var m Match
    query := `
        SELECT id, home_team, away_team, match_date,
        goals, yellow_cards, red_cards, 
        extra_time
        FROM matches
        WHERE id = $1
    `
    err = db.QueryRow(query, id).Scan(
        &m.ID,
        &m.HomeTeam,
        &m.AwayTeam,
        &matchDate,
        &m.Goals,
        &m.YellowCards,
        &m.RedCards,
        &m.ExtraTime,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Partido no encontrado", http.StatusNotFound)
        } else {
            http.Error(w, "Error al obtener el partido", http.StatusInternalServerError)
        }
        log.Println(err)
        return
    }

    m.MatchDate = matchDate.Format("2006-01-02")

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(m)
}

// Handler para hacer PUT de un partido
func putMatchHandler(w http.ResponseWriter, r *http.Request) {
    enableCors(w)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodPut {
        http.Error(w, "Método incorrecto utilizado", http.StatusMethodNotAllowed)
        return
    }

    idStr := strings.TrimPrefix(r.URL.Path, "/api/matches/")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "ID de partido inválido", http.StatusBadRequest)
        return
    }

    // Maneja matchDate como string, al igual que en postMatchHandler
    var updatedData struct {
        HomeTeam  string `json:"homeTeam"`
        AwayTeam  string `json:"awayTeam"`
        MatchDate string `json:"matchDate"`
    }
    if err := json.NewDecoder(r.Body).Decode(&updatedData); err != nil {
        http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
        log.Println("Error decoding JSON:", err)
        return
    }

    connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        log.Println("Error conectando a la DB:", err)
        return
    }
    defer db.Close()

    query := `
        UPDATE matches 
        SET home_team = $1, away_team = $2, match_date = $3
        WHERE id = $4
    `
    res, err := db.Exec(query, updatedData.HomeTeam, updatedData.AwayTeam, updatedData.MatchDate, id)
    if err != nil {
        http.Error(w, "Error al actualizar el partido", http.StatusInternalServerError)
        log.Println("Error en UPDATE:", err)
        return
    }

    rowsAffected, err := res.RowsAffected()
    if err != nil {
        http.Error(w, "Error al verificar actualización", http.StatusInternalServerError)
        log.Println("Error obteniendo filas afectadas:", err)
        return
    }
    if rowsAffected == 0 {
        http.Error(w, "Partido no encontrado", http.StatusNotFound)
        return
    }

    response := map[string]interface{}{
        "status":  "success",
        "message": fmt.Sprintf("Partido con ID %d actualizado", id),
        "id":      id,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Handler para hacer DELETE de un partido
func deleteMatchHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}


    if r.Method != http.MethodDelete {
        http.Error(w, "Método incorrecto utilizado", http.StatusMethodNotAllowed)
        return
    }

	// Obtener el ID del partido de la URL
    idStr := strings.TrimPrefix(r.URL.Path, "/api/matches/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID de partido inválido", http.StatusBadRequest)
		return
	}

    connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        log.Println("Error conectando a la DB:", err)
        return
    }
    defer db.Close()

    // Ejecutar el DELETE
    query := "DELETE FROM matches WHERE id = $1"
    res, err := db.Exec(query, id)
    if err != nil {
        http.Error(w, "Error al eliminar el partido", http.StatusInternalServerError)
        log.Println("Error ejecutando DELETE:", err)
        return
    }

    rowsAffected, err := res.RowsAffected()
    if err != nil {
        http.Error(w, "Error al verificar la eliminación", http.StatusInternalServerError)
        log.Println("Error obteniendo RowsAffected:", err)
        return
    }
    if rowsAffected == 0 {
        http.Error(w, "Partido no encontrado", http.StatusNotFound)
        return
    }

    // Enviar respuesta de éxito
    w.Header().Set("Content-Type", "application/json")
    response := map[string]interface{}{
        "status":  "success",
        "message": fmt.Sprintf("Partido con ID %d eliminado", id),
    }
    json.NewEncoder(w).Encode(response)
}

// Handler para hacer patch a algun partido especifico, para actualizar los goles
func patchMatchGoalHandler(w http.ResponseWriter, r *http.Request, id int) {
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodPatch {
        http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
        return
    }

    connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        return
    }
    defer db.Close()

    query := `
        UPDATE matches
        SET goals = goals + 1
        WHERE id = $1
    `
    res, err := db.Exec(query, id)
    if err != nil {
        http.Error(w, "Error al actualizar goles", http.StatusInternalServerError)
        return
    }

    rowsAffected, err := res.RowsAffected()
    if err != nil {
        http.Error(w, "Error al verificar la actualización", http.StatusInternalServerError)
        return
    }
    if rowsAffected == 0 {
        http.Error(w, "Partido no encontrado", http.StatusNotFound)
        return
    }

    //Jalar cantidad de goles
    query = "SELECT goals FROM matches WHERE id = $1"
    var goals int
    err = db.QueryRow(query, id).Scan(&goals)
    if err != nil {
        http.Error(w, "Error al obtener la cantidad de goles", http.StatusInternalServerError)
        return
    }

    // Respuesta
    response := map[string]interface{}{
        "status":  "success",
        "message": fmt.Sprintf("Partido con ID %d actualizado con %d goles", id, goals),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Handler para hacer patch a algun partido especifico, para actualizar las tarjetas amarillas
func patchMatchYellowCardHandler(w http.ResponseWriter, r *http.Request, id int) {
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodPatch {
        http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
        return
    }

    connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        return
    }
    defer db.Close()

    query := `
        UPDATE matches
        SET yellow_cards = yellow_cards + 1
        WHERE id = $1
    `
    res, err := db.Exec(query, id)
    if err != nil {
        http.Error(w, "Error al actualizar tarjetas amarillas", http.StatusInternalServerError)
        return
    }

    rowsAffected, err := res.RowsAffected()
    if err != nil {
        http.Error(w, "Error al verificar la actualización", http.StatusInternalServerError)
        return
    }
    if rowsAffected == 0 {
        http.Error(w, "Partido no encontrado", http.StatusNotFound)
        return
    }

    //Jalar cantidad de tarjetas amarillas
    var yellowCards int
    query = "SELECT yellow_cards FROM matches WHERE id = $1"
    err = db.QueryRow(query, id).Scan(&yellowCards)
    if err != nil {
        http.Error(w, "Error al obtener la cantidad de tarjetas amarillas", http.StatusInternalServerError)
        return
    }

    // Respuesta
    response := map[string]interface{}{
        "status":  "success",
        "message": fmt.Sprintf("Tarjetas amarillas incrementadas en el partido con ID %d. Cantidad actual: %d", id, yellowCards),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Handler para patch las tarjetas rojas
func patchMatchRedCardHandler(w http.ResponseWriter, r *http.Request, id int) {
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodPatch {
        http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
        return
    }

    connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        return
    }
    defer db.Close()

    query := `
        UPDATE matches
        SET red_cards = red_cards + 1
        WHERE id = $1
    `
    res, err := db.Exec(query, id)
    if err != nil {
        http.Error(w, "Error al actualizar tarjetas rojas", http.StatusInternalServerError)
        return
    }

    rowsAffected, err := res.RowsAffected()
    if err != nil {
        http.Error(w, "Error al verificar la actualización", http.StatusInternalServerError)
        return
    }
    if rowsAffected == 0 {
        http.Error(w, "Partido no encontrado", http.StatusNotFound)
        return
    }

    // Jalar la cantidad de tarjetas rojas
    var redCards int
    err = db.QueryRow("SELECT red_cards FROM matches WHERE id = $1", id).Scan(&redCards)
    if err != nil {
        http.Error(w, "Error al obtener la cantidad de tarjetas rojas", http.StatusInternalServerError)
        return
    }

    // Respuesta
    response := map[string]interface{}{
        "status":  "success",
        "message": fmt.Sprintf("Tarjetas rojas incrementadas en el partido con ID %d. Cantidad actual de tarjetas rojas: %d", id, redCards),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Handler para actualizar la cantidad de tiempo 
func patchMatchExtraTimeHandler(w http.ResponseWriter, r *http.Request, id int) {
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodPatch {
        http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
        return
    }

    connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        http.Error(w, "Error de conexión a la base de datos", http.StatusInternalServerError)
        return
    }
    defer db.Close()

    query := `
        UPDATE matches
        SET extra_time = extra_time + 1
        WHERE id = $1
    `
    res, err := db.Exec(query, id)
    if err != nil {
        http.Error(w, "Error al actualizar tiempo extra", http.StatusInternalServerError)
        return
    }

    rowsAffected, err := res.RowsAffected()
    if err != nil {
        http.Error(w, "Error al verificar la actualización", http.StatusInternalServerError)
        return
    }
    if rowsAffected == 0 {
        http.Error(w, "Partido no encontrado", http.StatusNotFound)
        return
    }

    // Obtener cantidad de tiempo extra
    var extraTime int
    err = db.QueryRow("SELECT extra_time FROM matches WHERE id = $1", id).Scan(&extraTime)
    if err != nil {
        http.Error(w, "Error al obtener la cantidad de tiempo extra", http.StatusInternalServerError)
        return
    }

    // Respuesta
    response := map[string]interface{}{
        "status":  "success",
        "message": fmt.Sprintf("Tiempo extra incrementado en el partido con ID %d. Tiempo extra actual: %d", id, extraTime),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func matchesHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        getMatchesHandler(w, r)
    case http.MethodPost:
        postMatchHandler(w, r)
    default:
        http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
    }
}


func main() {
    mux := http.NewServeMux()

    err := godotenv.Load()
    if err != nil {
        log.Println("No se pudo cargar el archivo .env")
    }

    // Endpoint para ping
    mux.HandleFunc("/ping", pingHandler)
    
    // Endpoint para obtener la lista de partidos y crear uno 
    mux.HandleFunc("/api/matches", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
            getMatchesHandler(w, r)
        case http.MethodPost:
            postMatchHandler(w, r)
        default:
            http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
        }
    })
    
    // Endpoint para operaciones sobre un partido específico:
    // para GET por ID, PUT y DELETE.
    mux.HandleFunc("/api/matches/", func(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/api/matches/")
        parts := strings.Split(path, "/")
    
        // Caso general: /api/matches/xfxx{id}
        if len(parts) == 1 {
            switch r.Method {
            case http.MethodGet:
                getMatchByIDHandler(w, r)
            case http.MethodPut:
                putMatchHandler(w, r)
            case http.MethodDelete:
                deleteMatchHandler(w, r)
            default:
                http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
            }
            return
        }
    
        // Caso especial: /api/matches/{id}/alguna-operacion
        if len(parts) == 2 && r.Method == http.MethodPatch {
            id, err := strconv.Atoi(parts[0])
            if err != nil {
                http.Error(w, "ID de partido inválido", http.StatusBadRequest)
                return
            }
    
            switch parts[1] {
            case "goals":
                patchMatchGoalHandler(w, r, id)
            case "yellowcards":
                patchMatchYellowCardHandler(w, r, id)
            case "redcards":
                patchMatchRedCardHandler(w, r, id)
            case "extratime":
                patchMatchExtraTimeHandler(w, r, id)
            default:
                http.Error(w, "Operación no soportada", http.StatusNotFound)
            }
            return
        }
    
        // Si no coincide con nada
        http.Error(w, "Ruta o método no soportado", http.StatusNotFound)
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    // Envolver el mux con el middleware CORS
    handler := corsMiddleware(mux)

    log.Printf("Servidor escuchando en el puerto %s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, handler))
}