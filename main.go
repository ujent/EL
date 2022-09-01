package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"html"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"fmt"

	"log"

	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

const (
	host = "cluster0-biconnector.spnud.mongodb.net"
	port = "27015"
	user = "bipp_with_psw?source=admin"
	psw  = "7W1iHudDpVklirw6"
	db   = "sample_guides"

	//net auth for bipp server
	tlsKeyPath  = "./key.pem"
	tlsCertPath = "./certificate.pem"

	//There are 2 users in M10 cluster: bipp and bipp_with_psw
	M10user = "bipp"
	M10Cert = "X509-cert-4557831596589561519.pem" //cert of bipp user

	M10userWithPsw = "bipp_with_psw"
	M10UserPsw     = "7W1iHudDpVklirw6"

	M10port       = "27015"
	M10Host       = "cluster0-biconnector.spnud.mongodb.net"
	M10DB         = "TestDB"
	M10Collection = "Scores"

	certM1       = "X509-cert-1495296871186416546.pem"
	M1DB         = "sample_guides"
	M1user       = "user_1"
	M1Collection = "planets"
)

var M1URI = "mongodb+srv://cluster0.66xkrso.mongodb.net/?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&tlsCertificateKeyFile=" + "./" + certM1
var M10URI = "mongodb+srv://cluster0.spnud.mongodb.net/?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&tlsCertificateKeyFile=" + "./" + M10Cert

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/test", testAuth)

	log.Fatal(http.ListenAndServe(":6003", nil))

}

func testAuth(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	// for example, don't work right now
	// err := standartMongo(ctx)
	// if err != nil {
	// 	writeError(w, http.StatusInternalServerError, err)
	// 	return
	// }

	client, err := mongoClient(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	meta, err := client.Metadata()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Println("Meta: ", meta)

	writeJSON(w, http.StatusOK, meta)
}

func mongoClient(ctx context.Context) (*Mongo, error) {

	tlsKeyPEM, err := ioutil.ReadFile(tlsKeyPath)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	tlsCertPEM, err := ioutil.ReadFile(tlsCertPath)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	cert, err := tls.X509KeyPair(tlsCertPEM, tlsKeyPEM)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	tlsCfgKey := "custom"
	mysql.RegisterTLSConfig(tlsCfgKey, &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	})

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?tls=%s&allowCleartextPasswords=true", user, psw, host, port, db, tlsCfgKey)
	client, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Print("error creating a Mongo Client: ", err)
		return nil, err
	}

	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	if err = client.Ping(); err != nil {
		log.Print("error in connection: ", err.Error())
		return nil, err
	}

	return &Mongo{Client: client, Database: db}, nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {

	json, err := json.Marshal(payload)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(json)
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	w.Write([]byte(err.Error()))
}

func writeOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

type Mongo struct {
	Client   *sql.DB
	Database string
}

func standartMongo(ctx context.Context) error {
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)

	clientOptions := options.Client().
		ApplyURI(M10URI).
		SetServerAPIOptions(serverAPIOptions)

	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		log.Fatal(err)
	}

	defer client.Disconnect(ctx)

	collection := client.Database(M10DB).Collection(M10Collection)

	docCount, err := collection.CountDocuments(ctx, bson.D{})

	if err != nil {
		log.Fatal(err)
		return err
	}

	fmt.Println(docCount)

	return nil
}

// Metadata returns the metadata associated with Mongo connection object
func (mongo *Mongo) Metadata() ([]Dataset, error) {

	return mongoShallowSchema(mongo)
}

type Dataset struct {
	Name       string    `json:"name"`                 // name of the dataset
	FQN        string    `json:"fqn"`                  // fully qualified name
	CreateTime time.Time `json:"ctime,omitempty"`      // create time
	ModifyTime time.Time `json:"mtime,omitempty"`      // modify time
	Tables     []Table   `json:"tables"`               // array of tables of this dataset/database
	Schema     string    `json:"schemaName,omitempty"` // SnowFlake configuration member: data hierarchy in SnowFlake is: warehouse -> 1 or more databases -> 1 or more schemas -> 1 or many tables
}

// Table represents a table of a dataset/database for a provider
type Table struct {
	Name    string   `json:"name"`              // name of the table
	FQSN    string   `json:"fqsn"`              // Fully Qualified SQL Name of the table
	SN      string   `json:"sn"`                // SQL Name of the table
	Columns []Column `json:"columns,omitempty"` // array of columns of the table
}

// Column represents a column field of a dataset/database table for a provider
type Column struct {
	Name      string `json:"name"`                  // name of the column
	SN        string `json:"sn"`                    // SQL Name of the column
	BlingType string `json:"bling_type"`            // data type in bling namespace
	Type      string `json:"type"`                  // data type of the column
	CastType  string `json:"cast_type"`             // preferable casting data type for esoteric data types, refer: https://www.notion.so/bipp/Money-and-other-data-esoteric-data-types-f9016278f7b0434781884a5bdd0c01c8
	Desc      string `json:"description,omitempty"` // description of the column
}

func mongoShallowSchema(mongo *Mongo) (schema []Dataset, err error) {

	var (
		database string
		query    string
		client   = mongo.Client
		tables   *sql.Rows
	)
	database = mongo.Database
	datasets := make([]Dataset, 0)

	query = fmt.Sprintf("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA='%s'", database)
	tables, err = client.Query(query)
	if err != nil {
		log.Println(err)
		return
	}
	defer tables.Close()

	var d = Dataset{}
	d.FQN = "`" + database + "`"
	d.Name = database
	d.Tables = make([]Table, 0)

	var tableMutex sync.Mutex
	var wg sync.WaitGroup

	for tables.Next() {
		var (
			t         = Table{}
			tableName string
		)

		err = tables.Scan(&tableName)
		if err != nil {
			log.Println("error while getting tables: ", err)
			return
		}

		wg.Add(1)

		go func(t *Table, tables *[]Table) {
			defer wg.Done()

			t.Name = tableName
			t.FQSN = d.FQN + "." + "`" + tableName + "`"
			t.SN = d.FQN + "." + tableName

			tableMutex.Lock()
			*tables = append(*tables, *t)
			tableMutex.Unlock()
		}(&t, &d.Tables)

	}
	wg.Wait()

	datasets = append(datasets, d)
	schema = datasets

	return
}
