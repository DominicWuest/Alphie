package todo

import (
	"fmt"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var mockTodo Todo
var dbMock sqlmock.Sqlmock

func TestMain(m *testing.M) {
	db, mock, err := sqlmock.New()
	if err != nil {
		fmt.Printf("an error '%s' was not expected when opening a stub database connection", err)
		os.Exit(1)
	}

	mockTodo = Todo{
		DB: db,
	}
	dbMock = mock

	statusCode := m.Run()

	db.Close()

	os.Exit(statusCode)
}

func TestParseIds(t *testing.T) {
	tests := []struct {
		input          []string
		expectedOutput []string
		expectedError  error
	}{
		{[]string{"1"}, []string{"1"}, nil},
		{[]string{"1", "1"}, []string{"1"}, nil},
		{[]string{"1", "2"}, []string{"1", "2"}, nil},
		{[]string{"1", ",2"}, []string{"1", "2"}, nil},
		{[]string{"1,", "2"}, []string{"1", "2"}, nil},
		{[]string{"1", "2,"}, []string{"1", "2"}, nil},
		{[]string{"1", "", "2"}, []string{"1", "2"}, nil},
		{[]string{"1,", "", "2"}, []string{"1", "2"}, nil},
		{[]string{"1,", "", "2"}, []string{"1", "2"}, nil},
		{[]string{"1", ",", "2"}, []string{"1", "2"}, nil},
		{[]string{"1", "", ",", "2"}, []string{"1", "2"}, nil},

		{[]string{"a"}, nil, fmt.Errorf("invalid id format")},
		{[]string{"1,", ",2"}, nil, fmt.Errorf("invalid id format")},
		{[]string{",1", "2"}, nil, fmt.Errorf("invalid id format")},
		{[]string{"1", ",,", "2"}, nil, fmt.Errorf("invalid id format")},
		{[]string{"1", ",", "", ",", "2"}, nil, fmt.Errorf("invalid id format")},
	}

	for _, test := range tests {
		result, err := parseIds(test.input)
		assert.ElementsMatch(t, test.expectedOutput, result)
		assert.Equal(t, test.expectedError, err)
	}
}

func TestDeduplicate(t *testing.T) {
	tests := []struct {
		input          []string
		expectedOutput []string
	}{
		{[]string{}, []string{}},
		{[]string{"1"}, []string{"1"}},
		{[]string{"1", "2"}, []string{"1", "2"}},
		{[]string{"1", "1"}, []string{"1"}},
		{[]string{"1", "1", "2"}, []string{"1", "2"}},
		{[]string{"1", "1", "2", "2"}, []string{"1", "2"}},
	}

	for _, test := range tests {
		result := deduplicate(test.input)
		assert.ElementsMatch(t, test.expectedOutput, result)
	}
}

func TestCheckUserPresenceInDB(t *testing.T) {
	dbMock.ExpectBegin()

	dbMock.ExpectQuery(`SELECT id FROM todo.discord_user`).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("0"))

	dbMock.ExpectCommit()

	assert.Nil(t, mockTodo.checkUserPresence("0"))

	assert.Nil(t, dbMock.ExpectationsWereMet())
}

func TestCheckUserPresenceNotInDB(t *testing.T) {
	dbMock.ExpectBegin()

	dbMock.ExpectQuery(`SELECT id FROM todo.discord_user`).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	dbMock.ExpectExec(`INSERT INTO todo.discord_user`).
		WithArgs("0").
		WillReturnResult(sqlmock.NewResult(1, 1))

	dbMock.ExpectCommit()

	assert.Nil(t, mockTodo.checkUserPresence("0"))

	assert.Nil(t, dbMock.ExpectationsWereMet())
}

func TestCreateTask(t *testing.T) {
	taskId := 1

	dbMock.ExpectQuery(`INSERT INTO todo.task`).
		WithArgs("taskAuthor", "taskTitle", "taskDescription").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(taskId))

	result, err := mockTodo.CreateTask("taskAuthor", "taskTitle", "taskDescription")

	assert.Equal(t, taskId, result)
	assert.Nil(t, err)
	assert.Nil(t, dbMock.ExpectationsWereMet())
}

func TestGetUserTODOsEmpty(t *testing.T) {
	dbMock.ExpectQuery(`SELECT (.+) FROM todo.task AS t JOIN todo.x AS a`).
		WithArgs("userId").
		WillReturnRows(sqlmock.NewRows([]string{"id", "creator", "title", "description"}))

	items, err := mockTodo.getUserTODOs("userId", "x")

	assert.Empty(t, items)
	assert.Nil(t, err)
	assert.Nil(t, dbMock.ExpectationsWereMet())
}

func TestGetUserTODOsNonEmpty(t *testing.T) {
	dbMock.ExpectQuery(`SELECT (.+) FROM todo.task AS t JOIN todo.x AS a`).
		WithArgs("userId").
		WillReturnRows(sqlmock.NewRows([]string{"id", "creator", "title", "description"}).AddRow(0, "c0", "t0", "d0").AddRow(1, "c1", "t1", "d1"))

	items, err := mockTodo.getUserTODOs("userId", "x")

	assert.Equal(t, []todoItem{{0, "c0", "t0", "d0"}, {1, "c1", "t1", "d1"}}, items)
	assert.Nil(t, err)
	assert.Nil(t, dbMock.ExpectationsWereMet())
}

func TestChangeItemStatus(t *testing.T) {
	dbMock.ExpectQuery(`SELECT task FROM todo.x`).
		WithArgs("0", pq.Array([]string{"1", "2"})).
		WillReturnRows(sqlmock.NewRows([]string{"task"}).AddRow("1").AddRow("2"))

	dbMock.ExpectBegin()

	dbMock.ExpectExec(`DELETE FROM todo.x`).
		WithArgs("0", pq.Array([]string{"1", "2"})).
		WillReturnResult(sqlmock.NewResult(1, 2))

	dbMock.ExpectExec(`INSERT INTO todo.y`).
		WithArgs("0", pq.Array([]string{"1", "2"})).
		WillReturnResult(sqlmock.NewResult(1, 2))

	dbMock.ExpectCommit()

	err := mockTodo.changeItemsStatus("0", []string{"1", "2"}, "x", "y")

	assert.Nil(t, err)
}

func TestChangeItemStatusWrongIDs(t *testing.T) {

	dbMock.ExpectQuery(`SELECT task FROM todo.x`).
		WithArgs("0", pq.Array([]string{"1", "2"})).
		WillReturnRows(sqlmock.NewRows([]string{"task"}).AddRow("1"))

	err := mockTodo.changeItemsStatus("0", []string{"1", "2"}, "x", "y")

	assert.IsType(t, &InvalidIDError{}, err)
}

func TestChangeItemStatusAllWrongIDs(t *testing.T) {
	dbMock.ExpectQuery(`SELECT task FROM todo.x`).
		WithArgs("0", pq.Array([]string{"1", "2"})).
		WillReturnRows(sqlmock.NewRows([]string{"task"}))

	err := mockTodo.changeItemsStatus("0", []string{"1", "2"}, "x", "y")

	assert.IsType(t, &InvalidIDError{}, err)
}
