/*

USAGE:

var db *DB
func main(){
  db = NewDB("user","pass","db")
  count := db.SelectInt("SELECT COUNT(*) FROM `user`")
  write( count )
}

*/
package main
import(
  "database/sql"
  "log"
  "time"
  _ "github.com/go-sql-driver/mysql"
)


type DB struct{
  db *sql.DB 
}


func NewDB(user, pass, database string) *DB {
	return &DB{
		db: InitDB(user, pass, database),
	}
}

func (db *DB) Insert(query string, args ...interface{}) int {
	return InsertDB(db.db, query, args...)
}

func (db *DB) Update(query string, args ...interface{}) int {
	return UpdateDB(db.db, query, args...)
}

func (db *DB) Delete(query string, args ...interface{}) int {
	return DeleteDB(db.db, query, args...)
}

func (db *DB) SelectInt(query string, args ...interface{}) int {
	return SelectIntDB(db.db, query, args...)
}

func (db *DB) SelectFloat(query string, args ...interface{}) float64 {
	return SelectFloatDB(db.db, query, args...)
}

func (db *DB) SelectString(query string, args ...interface{}) string {
	return SelectStringDB(db.db, query, args...)
}

func (db *DB) SelectInts(query string, args ...interface{}) []int {
	return SelectIntsDB(db.db, query, args...)
}

func (db *DB) SelectStrings(query string, args ...interface{}) []string {
	return SelectStringsDB(db.db, query, args...)
}

func (db *DB) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}


func InitDB(user,pass,database string) *sql.DB{
  db, err := sql.Open(
    "mysql", 
    user +":"+pass+"@unix(/run/mysqld/mysqld.sock)/"+database+"?parseTime=true",
  )
  db.SetMaxOpenConns(50)
  db.SetMaxIdleConns(50)
  db.SetConnMaxLifetime(5 * time.Minute)
  if err != nil {
		log.Fatal(err)
  }
  return db
}

func InsertDB(db *sql.DB, query string, args ...interface{}) int {
	res, err := db.Exec(query, args...)
	if err != nil {
		log.Println("Insert error:", err, "Query:", query, "Args:", args)
		return 0
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Println("LastInsertId error:", err)
		return 0
	}

	return int(id)
}

func UpdateDB(db *sql.DB, query string, args ...interface{}) int {
	res, err := db.Exec(query, args...)
	if err != nil {
		log.Println("Update error:", err, "Query:", query, "Args:", args)
		return 0
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Println("RowsAffected error:", err)
		return 0
	}

	return int(rows)
}

func DeleteDB(db *sql.DB, query string, args ...interface{}) int {
	res, err := db.Exec(query, args...)
	if err != nil {
		log.Println("Delete error:", err, "Query:", query, "Args:", args)
		return 0
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Println("RowsAffected error:", err)
		return 0
	}

	return int(rows)
}

func SelectIntDB(db *sql.DB, query string, args ...interface{}) int {
	var value int

	err := db.QueryRow(query, args...).Scan(&value)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println("SelectInt error:", err, "Query:", query, "Args:", args)
		}
		return 0
	}

	return value
}

func SelectFloatDB(db *sql.DB, query string, args ...interface{}) float64 {
	var value float64

	err := db.QueryRow(query, args...).Scan(&value)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println("SelectFloat error:", err, "Query:", query, "Args:", args)
		}
		return 0
	}

	return value
}

func SelectStringDB(db *sql.DB, query string, args ...interface{}) string {
	var value string

	err := db.QueryRow(query, args...).Scan(&value)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println("SelectString error:", err, "Query:", query, "Args:", args)
		}
		return ""
	}

	return value
}

func SelectIntsDB(db *sql.DB, query string, args ...interface{}) []int {
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("SelectInts error:", err, "Query:", query, "Args:", args)
		return []int{}
	}
	defer rows.Close()

	var results []int

	for rows.Next() {
		var value int

		if err := rows.Scan(&value); err != nil {
			log.Println("SelectInts scan error:", err)
			continue
		}

		results = append(results, value)
	}

	if err := rows.Err(); err != nil {
		log.Println("SelectInts rows error:", err)
	}

	return results
}

func SelectStringsDB(db *sql.DB, query string, args ...interface{}) []string {
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("SelectStrings error:", err, "Query:", query, "Args:", args)
		return nil
	}
	defer rows.Close()

	var values []string

	for rows.Next() {
		var value string

		if err := rows.Scan(&value); err != nil {
			log.Println("SelectStrings scan error:", err)
			return nil
		}

		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		log.Println("SelectStrings rows error:", err)
		return nil
	}

	return values
}
