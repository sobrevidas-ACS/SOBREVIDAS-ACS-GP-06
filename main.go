package main

import (
	"database/sql"
	// "fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	

	dbPostgres, err := connectDBPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer dbPostgres.Close()

	_, err = db.Exec("INSERT INTO login (usuario, senha) VALUES ($1, $2)", "ana", "123")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/style/", http.StripPrefix("/style/", http.FileServer(http.Dir("/style"))))

	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.ServeFile(w, r, "templates/login.html")
	})

	http.HandleFunc("/welcome", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.ServeFile(w, r, "templates/welcome.html")
	})

	http.HandleFunc("/cadastro", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			cadastroHandler(w, r, dbPostgres)
		} else {
			w.Header().Set("Content-Type", "text/html")
			http.ServeFile(w, r, "templates/cadastro.html")
		}
	})

	http.HandleFunc("/patients", func(w http.ResponseWriter, r *http.Request) {
		patientsHandler(w, r, dbPostgres)
	})

	http.HandleFunc("/delete-patient", func(w http.ResponseWriter, r *http.Request) {
		deletePatientHandler(w, r, dbPostgres)
	})

	http.HandleFunc("/login", loginHandler(db))

	log.Fatal(http.ListenAndServe(":8080", nil))
	
}


func loginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		var errorMessage string
		if username == "" || password == "" {
			errorMessage = "empty_credentials"
			http.Redirect(w, r, "/?error="+errorMessage, http.StatusFound)
			return
		}

		var storedPassword string
		err := db.QueryRow("SELECT senha FROM public.login WHERE usuario = $1", username).Scan(&storedPassword)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Invalid username or password", http.StatusUnauthorized)
				return
			}
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if password == storedPassword {
			http.Redirect(w, r, "/welcome", http.StatusFound)
			return
		}

		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
	}
}

func connectDB() (*sql.DB, error) {
	connStr := "user=lucas dbname=login sslmode=disable password=1234"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func connectDBPostgres() (*sql.DB, error) {
	connStr := "user=lucas dbname=postgres sslmode=disable password=1234"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

type Person struct {
	ID     int
	Nome   string
	CPF    string
	Idade  int
	Sexo   string
	Fuma   string
	Alcool string
}

func patientsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    busca := r.URL.Query().Get("busca")

    var rows *sql.Rows
    var err error

    if busca != "" {
        query := `
            SELECT * FROM pacientes
            WHERE nome ILIKE '%' || $1 || '%'
            ORDER by nome
        `
        rows, err = db.Query(query, busca)
    } else {
        rows, err = db.Query("SELECT * FROM pacientes ORDER by nome")
    }

    if err != nil {
        log.Printf("Erro ao consultar pacientes: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var patients []Person

    for rows.Next() {
        var p Person
        if err := rows.Scan(&p.ID, &p.Nome, &p.CPF, &p.Idade, &p.Sexo, &p.Fuma, &p.Alcool); err != nil {
            log.Printf("Erro ao escanear paciente: %v", err)
            continue
        }
        patients = append(patients, p)
    }
    /*if err = rows.Err(); err != nil {
        log.Printf("Erro após iterar as linhas: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    log.Printf("Pacientes encontrados: %v", patients)*/

    tmpl, err := template.ParseFiles("templates/patients.html")
    if err != nil {
        log.Printf("Erro ao carregar template HTML: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if err := tmpl.Execute(w, patients); err != nil {
        log.Printf("Erro ao executar template HTML: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}


func cadastroHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    nome := r.FormValue("nome")
    cpf := r.FormValue("cpf")
    idade, err := strconv.Atoi(r.FormValue("idade"))
    if err != nil {
        http.Error(w, "Idade inválida", http.StatusBadRequest)
        return
    }
    sexo := r.FormValue("sexo")
    fuma := r.FormValue("fuma")
    alcool := r.FormValue("alcool")

    newPerson := Person{Nome: nome, CPF: cpf, Idade: idade, Sexo: sexo, Fuma: fuma, Alcool: alcool}

    log.Printf("Registrando nova pessoa: %+v", newPerson)
    idInserido, err := Registrar(db, newPerson)
    if err != nil {
        log.Printf("Erro ao registrar o novo paciente: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    log.Printf("Novo paciente registrado com ID: %d", idInserido)


    http.Redirect(w, r, "/patients", http.StatusSeeOther)
}



func Registrar(db *sql.DB, p Person) (int, error) {
	var id int
	err := db.QueryRow("INSERT INTO pacientes(nome, CPF, Idade, Sexo, Fuma, Alcool) VALUES($1, $2, $3, $4, $5, $6) RETURNING id", p.Nome, p.CPF, p.Idade, p.Sexo, p.Fuma, p.Alcool).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func deletePatientHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    // Extract patient ID from request
    patientID := r.URL.Query().Get("id")

    // SQL statement to delete a patient
    stmt := `DELETE FROM patients WHERE id = $1;`

    // Execute the SQL statement
    _, err := db.Exec(stmt, patientID)
    if err != nil {
        // Handle error
        http.Error(w, "Failed to delete patient", http.StatusInternalServerError)
        return
    }

    // Redirect or send a response indicating success
    http.Redirect(w, r, "/patients", http.StatusSeeOther)
}