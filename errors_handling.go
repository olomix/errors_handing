package errors_handling

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
)

type statsRequest struct{}

// parse http handler request
func parseStatsRequest(r *http.Request) (statsRequest, error) {
	err := r.ParseForm()
	if err != nil {
		return statsRequest{}, errors.WithStack(err)
	}
	return statsRequest{}, nil
}

// parse http handler request
func parseLimitRequest(r *http.Request) (int32, error) {
	err := r.ParseForm()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	limit, err := strconv.ParseInt(r.Form.Get("limit"), 10, 32)
	return int32(limit), errors.WithStack(err)
}

func handleErr(err error) {
	switch e := errors.Cause(err).(type) {
	case *pgconn.PgError:
		log.Printf("database error with code %v happened: %+v", e.Code, err)
	default:
		log.Printf("%+v", err)
	}
}

type ServerStats struct{}

// GetServerStats is a buisness method thant returns server stats
func GetServerStats(ctx context.Context, req statsRequest) (ServerStats, error) {
	return ServerStats{}, nil
}

func writeJSON(ctx context.Context, w http.ResponseWriter, statusCode int,
	v interface{}) {

	select {
	case <-ctx.Done():
		// соединение прервано, никто больше не ждет нашего ответа.
		return
	default:
	}

	data, err := json.Marshal(v)
	if err != nil {
		// Здесь мы не смогли замаршалить ответ. Это внутренняя ошибка сервера.
		// Мы должны залогировать ее со stacktrace'ом, чтоб понимать в каком
		// хендлере она произошла. Но клиенту мы ответим общей фразой, не
		// содержащей никаких внутренних названий полей или другой критичной
		// информации
		log.Printf("%+v", errors.WithStack(err))
		http.Error(w, "unable to encode response to json",
			http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		// А вот эту ошибку в большинстве случаев можно игнорировать.
		// Если мы добавим сюда логирование, то будем получать ненужные нам
		// нам ошибки, что клиент прервал соединение. Хотя возможно
		// здесь имеет смысл добавить метрику, чтоб можно было настроить алерты
		// на случай если подозрительно много клиентов станет рвать соединение.
		// Это может быть проблема с нашим ответом либо попытка какой-то атаки.
	}
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	statsRequest, err := parseStatsRequest(r)
	if err != nil {
		// Если возникла проблема с запросом, это клиент виноват, что
		// не верно сформировал запрос. Нам не нужен stacktrace в логах.
		// Достаточно будет обычного лога от middleware, что был 400-й ответ.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Бизнес-методы, как правило, принимают context. Если клиент прервет
	// соединение, они смогут завершить свою работу, которая уже никому не
	// нужна.
	stats, err := GetServerStats(ctx, statsRequest)
	if err != nil {
		// Здесь мы уже логируем ошибку со stacktrace'ом, т.к. это внутренняя
		// ошибка сервера и ее нужно исправлять. Отдельно можно обрабатывать
		// ошибки от закрытого контекста. Но я предпочитаю этого не делать, т.к.
		// если произошел timeout и закрылся контекст, то stacktrace будет
		// содержать место, где возник timeout. Как правило, это какой-то долгий
		// запрос в базу и видно что нужно оптимизировать.
		log.Printf("%+v", err)
		// Здесь два варианта что возвращать клиенту. Если мы пишем какой-то
		// внутренний сервис, то безопасно всегда возвращать err.Error(). Но
		// если это какой-то публичный сервер, возможно имеет смысл писать
		// здеcь какую-то общую фразу server error, чтоб не экспоузить какие-то
		// кишки приложения наружу. Или держать набор возможных ошибок для
		// клиента с разными текстами.
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(ctx, w, http.StatusOK, stats)
}
