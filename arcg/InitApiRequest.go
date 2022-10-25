package arcg

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"

	"github.com/ecsaas/api-request-config/DEFINE_VARIABLES/arcgf"
	"github.com/ecsaas/api-request-config/DEFINE_VARIABLES/arcgt"
	"github.com/go-playground/validator"
)

func NewApiRequestConfig(request *http.Request, writer http.ResponseWriter) InitApiRequest { //arcg
	//Khởi tạo biến tạm cho luồng xử lý thông báo hệ thống và thông báo lỗi, status code put vào object lưu tạm để trả lại client qua hàm UnMound
	return InitApiRequest{ //arcg
		Request:     request,
		Writer:      writer,
		StatusCode:  &struct{ HttpCode int }{HttpCode: http.StatusOK},
		ServerAlert: &ServerAlert{},
		ErrorType:   &ErrorTypeList{},
		Store:       &Store{},
	}
}

type ObjectCheckError struct {
	//Object cấu hình các hàm xử lý biến check lỗi
	ConfigFieldCaseCheck   func(fieldError validator.FieldError) (fieldCaseCheck string) //arcg
	SwitchCaseCheckByField func(fieldCaseCheck string) (oer Error)                       //arcg
}

type InitApiRequest struct {
	//Object các tham số response phục vụ trả lại các data thông báo cho client
	//request
	Request *http.Request
	Writer  http.ResponseWriter
	//response
	StatusCode  *struct{ HttpCode int }
	ServerAlert *ServerAlert
	ErrorType   *ErrorTypeList
	//tmp
	Store *Store
}

func (a InitApiRequest) LoadAndParseData(dataParse interface{}) (check bool) { //arcg
	//Đọc data từ client và parse đổ dữ liệu vào biến được khai báo sử dụng
	var dataRequest, err = io.ReadAll(a.Request.Body)
	if err == nil {
		defer a.Request.Body.Close()
		if dataRequest != nil {
			check = json.Unmarshal(dataRequest, &dataParse) == nil
		}
	} else {
		*a.ServerAlert = ServerAlert{ //arcg
			Message: err.Error(),
			Code:    -123,
		}
	}
	if !check {
		a.BadRequest() //arcg
	}
	return
}

func (a InitApiRequest) UnMound(
	serverAlert *ServerAlert,
	errorType *ErrorTypeList,
	redirect bool,
	redirectUrl func() string,
) { //arcg
	//Trước khi hoàn thành vòng đời 1 request, sẽ thực hiện UnMound xử lý (response) trả lại header status code và các thông báo về client
	if redirect {
		http.Redirect(a.Writer, a.Request, redirectUrl(), http.StatusSeeOther)
	} else {
		a.Writer.WriteHeader(a.StatusCode.HttpCode)
	}
	if serverAlert != nil {
		*serverAlert = *a.ServerAlert
	}
	if errorType != nil {
		if a.ErrorType == nil {
			a.ErrorType = &ErrorTypeList{} //arcg
		}
		*errorType = *a.ErrorType
	}
}

func (a InitApiRequest) BadRequest() { //arcg
	//Gắn cờ (Check point) khi gặp lỗi và put data lỗi 400 vào Object InitApiRequest
	a.StatusCode.HttpCode = http.StatusBadRequest
	*a.ErrorType = ErrorTypeList{Error{ //arcg
		Field: arcgf.CLIENT_ERROR,
		Type:  arcgt.BAD_REQUEST,
	}}
}

func (a InitApiRequest) BadRequestErrorType(errorType ErrorTypeList) (exit bool) { //arcg
	//Gắn cờ (Check point) khi gặp lỗi và put data lỗi 400 kèm thông báo client vào Object InitApiRequest
	a.StatusCode.HttpCode = http.StatusBadRequest
	*a.ErrorType = errorType
	if len(errorType) > 0 {
		exit = true
	}
	return
}

func (a InitApiRequest) BadRequestServerAlert(message string, code int) { //arcg
	//Gắn cờ (Check point) khi gặp lỗi và put data lỗi 400 kèm thông báo server vào Object InitApiRequest
	a.StatusCode.HttpCode = http.StatusBadRequest
	*a.ErrorType = ErrorTypeList{Error{ //arcg
		Field: arcgf.CLIENT_ERROR,
		Type:  arcgt.BAD_REQUEST,
	}}
	*a.ServerAlert = ServerAlert{ //arcg
		Message: message,
		Code:    code,
	}
}

func (a InitApiRequest) StatusOK() { //arcg
	//Gắn cờ (Check point) hoàn thành request 200 (OK) vào Object InitApiRequest
	a.StatusCode.HttpCode = http.StatusOK
	*a.ErrorType = ErrorTypeList{} //arcg
}

func (a InitApiRequest) StatusCreated() { //arcg
	//Gắn cờ (Check point) hoàn thành request 201 (Khởi tạo) vào Object InitApiRequest
	a.StatusCode.HttpCode = http.StatusCreated
	*a.ErrorType = ErrorTypeList{} //arcg
}

func (a InitApiRequest) Unauthorized() { //arcg
	//Gắn cờ (Check point) khi gặp lỗi và put data lỗi 401 (lỗi token đăng nhập, hoặc password sai) vào Object InitApiRequest
	a.StatusCode.HttpCode = http.StatusUnauthorized
	*a.ErrorType = ErrorTypeList{Error{ //arcg
		Field: arcgf.TOKEN_ERROR,
		Type:  arcgt.TOKEN,
	}}
}

func (a InitApiRequest) BadGateway(message string, code int) { //arcg
	//Gắn cờ (Check point) khi gặp lỗi và put data lỗi 502 (lỗi nhiều vấn đề từ server) kèm thông báo server vào Object InitApiRequest
	a.StatusCode.HttpCode = http.StatusBadGateway
	*a.ErrorType = ErrorTypeList{Error{ //arcg
		Field: arcgf.SERVER_ERROR,
		Type:  arcgt.BAD_GATEWAY,
	}}
	if code > -1 {
		*a.ServerAlert = ServerAlert{ //arcg
			Message: message,
			Code:    code,
		}
	}
}

func (a InitApiRequest) ValidateRequestDataByOCE(
	oce ObjectCheckError,
	sc interface{},
) (errorType ErrorTypeList) { //arcg
	//Hàm thực hiện quét lỗi và put lỗi vào mảng lỗi tạm ErrorTypeList qua các tham số được cấu hình ở oce ObjectCheckError
	var validate = validator.New()
	var err = validate.Struct(sc)
	if _, ok := err.(*validator.InvalidValidationError); ok {
		errorType = append(errorType, Error{ //arcg
			Field: arcgf.SERVER_ERROR,
			Type:  arcgt.BAD_GATEWAY,
		})
		return
	}
	if err != nil {
		for _, errFields := range err.(validator.ValidationErrors) { //arcg
			var _errorType = oce.SwitchCaseCheckByField(oce.ConfigFieldCaseCheck(errFields)) //arcg
			if len(_errorType.Type) > 0 && len(_errorType.Field) > 0 {
				errorType = append(errorType, _errorType)
			}
		}
	}
	return
}

func (a InitApiRequest) ValidateSpecialPassword(
	password string,
	_Field string,
	_ErrorType string,
) (errorType ErrorTypeList) { //arcg
	//Hàm check điều kiện cho password phải đủ các điều kiện có ít nhất (1 upper, 1 lower, 1 number, 1 ký tự đặc biệt)
	//và put lỗi chung _ErrorType vào mảng nếu thiếu 1 trong số các điều kiện trên sau đó break
	errorType = ErrorTypeList{} //arcg
	for _, regexCheck := range []string{
		REGEXP_LOWER_AZ,
		REGEXP_UPPER_AZ,
		REGEXP_NUMBER,
		REGEXP_SPECIAL_CHARACTER,
	} {
		matched, _ := regexp.MatchString(regexCheck, password)
		if !matched {
			errorType = ErrorTypeList{ //arcg
				Error{ //arcg
					Field: _Field,
					Type:  _ErrorType,
				},
			}
			break
		}
	}
	return
}
