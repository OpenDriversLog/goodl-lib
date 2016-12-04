package models
/*
JSON CRUD-answers.
 */

// JSONAnswer is a basic JSON-CRUD-answer able to provide errors and tell if the query was successful.
type JSONAnswer struct {
	Success      bool
	Error        bool
	ErrorMessage string
	Errors       map[string]string
}

// JSONSelectAnswer is the default answer to a select/read query, including a result.
type JSONSelectAnswer struct {
	JSONAnswer
	Result interface{}
}

// JSONInsertAnswer is the default answer to a insert/create-query, including the key of the created item.
type JSONInsertAnswer struct {
	JSONAnswer
	LastKey int64
}

// JSONDeleteAnswer is the default answer to a delete-query, including the count of deleted rows and the id that was deleted.
type JSONDeleteAnswer struct {
	JSONAnswer
	RowCount int64
	Id int64
}

// JSONUpdateAnswer is the default answer to an update-query, including the count of updated rows and the id that was updated.
type JSONUpdateAnswer struct {
	JSONAnswer
	RowCount int64
	Id int64
}

// GetBadJSONInsertAnswer returns a bad JSONInsertAnswer in case of a failed insert/create-query.
func GetBadJSONInsertAnswer(message string) (ans JSONInsertAnswer) {
	return JSONInsertAnswer{
		JSONAnswer: GetBadJSONAnswer(message),
		LastKey:    -1,
	}
}

// GetGoodJSONInsertAnswer returns a good JSONInsertAnswer in case of a succesful insert/create-query.
func GetGoodJSONInsertAnswer(lastKey int64) (ans JSONInsertAnswer) {
	return JSONInsertAnswer{
		JSONAnswer: GetGoodJSONAnswer(),
		LastKey:    lastKey,
	}
}

// GetBadJSONSelectAnswer returns a bad JSONSelectAnswer in case of a failed select/read-query.
func GetBadJSONSelectAnswer(message string) (ans JSONSelectAnswer) {
	return JSONSelectAnswer{
		JSONAnswer: GetBadJSONAnswer(message),
	}
}

// GetGoodJSONSelectAnswer returns a good JSONSelectAnswer in case of a succesful select/read-query.
func GetGoodJSONSelectAnswer(result interface{}) (ans JSONSelectAnswer) {
	return JSONSelectAnswer{
		JSONAnswer: GetGoodJSONAnswer(),
		Result:     result,
	}
}

// GetBadJSONDeleteAnswer returns a bad JSONDeleteAnswer in case of a failed delete-query.
func GetBadJSONDeleteAnswer(message string, id int64) (ans JSONDeleteAnswer) {
	return JSONDeleteAnswer{
		JSONAnswer: GetBadJSONAnswer(message),
		RowCount:   -1,
		Id: id,
	}
}

// GetGoodJSONDeleteAnswer returns a good JSONDeleteAnswer in case of a successful delete-query.
func GetGoodJSONDeleteAnswer(rowCount int64, id int64) (ans JSONDeleteAnswer) {
	return JSONDeleteAnswer{
		JSONAnswer: GetGoodJSONAnswer(),
		RowCount:   rowCount,
		Id: id,
	}
}

// GetBadJSONUpdateAnswer returns a bad JSONUpdateAnswer in case of a failed update-query.
func GetBadJSONUpdateAnswer(message string,id int64) (ans JSONUpdateAnswer) {
	return JSONUpdateAnswer{
		JSONAnswer: GetBadJSONAnswer(message),
		RowCount:   -1,
		Id: id,
	}
}

// GetGoodJSONUpdateAnswer returns a good JSONUpdateAnswer in case of a successful update-query.
func GetGoodJSONUpdateAnswer(rowCount int64, id int64) (ans JSONUpdateAnswer) {
	return JSONUpdateAnswer{
		JSONAnswer: GetGoodJSONAnswer(),
		RowCount:   rowCount,
		Id: id,
	}
}
// GetBadJSONAnswer returns a bad JSONAnswer in case of an error/failed query.
func GetBadJSONAnswer(message string) (ans JSONAnswer) {
	return JSONAnswer{
		Error:        true,
		ErrorMessage: message,
	}
}

// GetGoodJSONAnswer returns a good JSONAnswer in case of a successful query.
func GetGoodJSONAnswer() (ans JSONAnswer) {
	return JSONAnswer{
		Success: true,
	}
}
