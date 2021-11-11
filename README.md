# Работа с ошибками в Go

> Этот документ является актуальным только при работе над 
> приложением типа web/grpc сервиса или командной утилитой.
> Он не предполагает работу над библиотекой. В библиотеках
> ошибки обрабатываются по другому.

Когда мы работаем с ошибками, мы ходим понимать причину их 
возникновения. Когда в программе возникает ошибка, мы хотим
знать две вещи:

1. Место в коде, где возникла ошибка. Одну и ту-же ошибку
   могут вернуть из разных мест. Поэтому важно понимать
   откуда именно вернули ошибку.
2. Stack trace к этому месту. Если ошибка возникла в каком-то
   утилитном методе, который парсит строку в число, нам
   важно знать какой код вызвал этот метод, чтоб исправить
   ошибку.

Оба пункта будут удовлетворены, если мы сможем распечатать
ошибку вместе со stacktrace'ом к месту ее возникновения.

Чтоб создавать ошибки, которые несут в себе stacktrace'ы, есть
модуль `github.com/pkg/errors`.

В нем есть методы, которые позволяют создать новую ошибку
со стектрейсом:
* [`errors.New(message string) error`](https://pkg.go.dev/github.com/pkg/errors#New)
* [`errors.Errorf(format string, args ...interface{}) error`](https://pkg.go.dev/github.com/pkg/errors#Errorf)

А так-же методы, позволяющие обернуть существующую ошибку 
чтоб она содержала stacktrace к тому месту, где происходит
это оборачивание.
* [`errors.WithStack(err error) error`](https://pkg.go.dev/github.com/pkg/errors#WithStack)
* [`errors.Wrap(err error, message string) error`](https://pkg.go.dev/github.com/pkg/errors#Wrap)
* [`errors.Wrapf(err error, format string, args ...interface{}) error`](https://pkg.go.dev/github.com/pkg/errors#Wrapf)

Если вы обернули ошибку с помощью одного из этих методов,
ошибка будет содержать stacktrace к месту где вы сделали
это оборачивание. Т.е. к месту вызова одной из функций `WithStack`,
`Wrap` или `Wrapf`.

Если ошибка уже содержала stacktrace до этого (т.е. уже была
создана с помощью библиотеки `github.com/pkg/errors`), то
в ней будут два отдельных stacktrace'а. И старый и новый. И при
логирование распечатаются оба. Понять где закончился один и
начался другой stacktrace сложно, поэтому лучше так не делать.
Чтоб избежать двойного оборачивания, нужно пользоваться тремя
простыми правилами при создании и возврате ошибок.

## Правила как создавать и оборачивать ошибки.

1. Если вы создаете новую ошибку, ее обязательно нужно создавать
   с помощью одной из функций библиотеки `github.com/pkg/errors`
   `New` или `Errorf`. Тогда она будет содержать stacktrace к 
   месту создания.
2. Если вы получаете ошибку от сторонней библиотеки или из 
   стандартной библиотеки (из либой библиотеки внешней для вашей
   программы). Тогда ее нужно обернуть с помощью
   одной из функций `WithStack`, `Wrap` или `Wrapf`. Тогда 
   вы сможете видеть stacktrace от места входа в код до места
   где вы получили ошибку из внешней библиотеки.
3. Если вы получаете ошибку из функции в этом-же проекте,
   с ней не нужно ничего делать. Просто можно возвращать как есть.
   Если придерживаться правил 1 и 2, то ошибка в себе уже содержит
   stacktrace к месту возникноверния и кто-то выше ее обработает
   или залогирует.

## Логирование ошибок

Логирование ошибки стоит делать только на самом верхнем
уровне. Если есть возможность вернуть ошибку тому, кто
нас вызвал, лучше вернуть ошибку. Если нет возможности
ее вернуть, можно ее залогировать.

Отдельный пункт это ошибки, которые мы обрабатываем.
Например мы можем ожидать ошибку timeout'а и попробуем
несколько раз повторить попытку перед тем как вернуть 
ошибку выше. В этом случае мы можем логировать ошибку. 
Но это никак не противоречит утверждению выше. Мы не 
можем вернуть ошибку timeout'а, т.к. мы собираемся ее
проигнорировать и повторить попытку. Поэтому "если нет
возможности вернуть, можно ее залогировать", остается 
верным.

Чтоб залогировать ошибку вместе со stacktrace'ом, нужно 
добавить `+` в format verb'у. Например
```go
log.Printf("%+v", err)
```
Если указать format verb без спецификатора `+`, выведется тект
ошибки, который возвращается из `err.Error()` без stacktrace'а.

### Ошибки в worker'ах.

Отдельно стоит упомянуть логирование ошибок во внутренних
background worker'ах. Т.е. если у вас есть какая-то goroutine с 
циклом, которая либо что-то периодически делает, либо 
обрабатывает какой-то канал, то вполне нормально логировать
в ней ошибки, т.к. она никуда их не может вернуть. 

### Логирование ошибок в HTTP запросах.

Удобно видель логи всех запросов к вашему web-серверу. Поэтому,
как правило, многие люди добавляют какой-то middleware к своему
http router'у, который может логировать URL запроса, status code, 
возможно request/response size и другие параметры. Но такие
middleware не имеют доступа к ошибкам, которые возникают в 
процессе обработки запроса и не могут залогировать их со
stacktrace'ом.

Поэтому http handler логирует ошибку со stacktrace'ом отдельно.
Например
```go
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(ctx, w, http.StatusOK, stats)
}
```

Здесь есть некая проблема, которая не будет здесь освещена.
У вас есть нагруженный веб-сервер, который пишет много логов. 
И вам бы хотелось объединять между собой ошибки, которые относятся
к одному запросу. Например вы видите запись от logging middleware,
что вы вернули 500-ю ошибку и хотели бы увидеть stacktrace этой
ошибки. Но он залогирован отдельным сообщением и найти его глазами
порой сложно. Для этого может применяться какой-то request-id.

Например, в сервере может быть какой-то `RequestIDMiddleware`,
который вытаскивает http header `X-Request-ID` из запроса и
помещает его в context запроса. Если такого хидера нет, 
генерируется случайный UUID и так само помещается в ctx. 
Затем библиотека логирования настраивается так, чтоб все 
логи аннотировались этим request-id, который берется из
контекста. Таким образом можно будет сгруппировать все логи 
по одному запросу и просмотреть их.

Более того этого request-id может пробрасываться в другие сервисы 
и можно будет видеть логи от разных сервисов, которые все относятся
к одному запросу. 

Но вся эта магия за пределами данной статьи.

## Обработка ошибок (Cause, As)

Если мы хотим обработать обернутую ошику, у нас есть несколько 
способов.

Изначально был метод `errors.Cause`. Он рекурсивно разворачивает
все обертывания, сделанные библиотекой `pkg/errors` пока не
доберется до оригинальной ошибки. Ее уже можно сравнивать с 
ошибкой, которую вы ожидаете. Например

```go
switch e := errors.Cause(err).(type) {
case *pgconn.PgError:
	log.Printf("database error with code %v happened: %+v", e.Code, err)
default:
	log.Printf("%+v", err)
}
```

В Go 1.13 добавили свои методы для работы с обернутыми ошибками.
И `pkg/errors` поддержал их в последующих версиях. Все-равно
стандартной библиотеки Go не достаточно чтоб выводить stacktrace'ы,
но мы можем использовать ее для обработки ошибок и доставания
"оригинальной" ошибки из обернутой. 

Пример.
```go
var err error = &pgconn.PgError{
	Code:    "101",
	Message: "error message",
}
err = errors.WithStack(err)
var err2 = new(pgconn.PgError)
if errors.As(err, &err2) {
	log.Printf("database error with code %v: %+v", err2.Code, err)
}
```

## Советы

### Оборачивание nil ошибок

Библиотека `pkg/errors` понимает если ей передать `nil` вместо
ошибки, что позволяет писать более простой код не проверяя 
постоянно `if err != nil`. Например, если мы хотим вызвать
стороннюю библиотеку и обернуть ошибку в stacktrace в случае
если она не `nil`, то можно сразу писать так

```go
func parseLimitRequest(r *http.Request) (int32, error) {
	err := r.ParseForm()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	limit, err := strconv.ParseInt(r.Form.Get("limit"), 10, 32)
	// Не нужно писать `if err != nil`. WithStack и так вернет nil.
	return int32(limit), errors.WithStack(err)
}
```

### Не злоупотребляйте Wrap & Wrapf

`Wrap` добавляет помимо stacktrace'а еще и текст к ошибке. 
Если вам нечего добавить, не нужно бессмысленно использовать Wrap.
Достаточно просто `WithStack`. По стеку и так будет понятно
почему произошла ошибка. 

