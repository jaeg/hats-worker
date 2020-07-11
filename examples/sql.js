db = sql.New("user:password@tcp(127.0.0.1:3306)/default", "mysql")
console.log(db.Ping())

rowsImpacted = db.Exec("Insert Into Test (Value1,Value2) VALUES (5,'hellooo')")
console.log("Rows impacted", rowsImpacted)

results = db.Query("SELECT * FROM Test")
console.log("Result count", results.length)
console.log(JSON.stringify(results))
db.Close()