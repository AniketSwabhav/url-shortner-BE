package security

import (
	"fmt"
	"net/http"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/web"
)

func MiddlewareAdmin(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		claim := Claims{}

		err := ValidateToken(w, r, &claim)
		if err != nil {
			fmt.Println("err=> ", err)
			web.RespondError(w, errors.NewValidationError("Invalid or missing token"))
			return
		}

		if !claim.IsAdmin {
			fmt.Println("User is not Admin")
			web.RespondError(w, errors.NewUnauthorizedError("Only admin can access this resource======================>"))
			return
		}

		if !claim.IsActive {
			fmt.Println("User is not Active")
			web.RespondError(w, errors.NewInActiveUserError("Only active admin can access this resource"))
			return
		}

		next.ServeHTTP(w, r)

	})
}

func MiddlewareUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		claim := Claims{}

		err := ValidateToken(w, r, &claim)
		if err != nil {
			fmt.Println("err=> ", err)
			web.RespondError(w, errors.NewValidationError("Invalid or missing token"))
			return
		}

		if claim.IsAdmin {
			fmt.Println("User is not Admin")
			web.RespondError(w, errors.NewUnauthorizedError("Only non-admin user can access this resource"))
			return
		}

		if !claim.IsActive {
			fmt.Println("User is not Active")
			web.RespondError(w, errors.NewInActiveUserError("Only active user can access this resource"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func MiddlewareCommon(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		claim := Claims{}

		err := ValidateToken(w, r, &claim)
		if err != nil {
			fmt.Println("err =>", err)
			web.RespondError(w, errors.NewValidationError("Invalid or missing token"))
			return
		}

		if !claim.IsActive {
			fmt.Println("User is not Active")
			web.RespondError(w, errors.NewInActiveUserError("Only active user can access this resource"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// func MiddlewareUrl(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

// 		claim := Claims{}

// 		err := ValidateToken(w, r, &claim)
// 		if err != nil {
// 			fmt.Println("err =>", err)
// 			web.RespondError(w, errors.NewValidationError("Invalid or missing token"))
// 			return
// 		}

// 		if !claim.IsActive {
// 			fmt.Println("User is not Active")
// 			web.RespondError(w, errors.NewInActiveUserError("Current user is not active"))
// 			return
// 		}

// 		if claim.IsAdmin {
// 			fmt.Println("admin cannot access user")
// 			web.RespondError(w, errors.NewUnauthorizedError("admin cannot access user urls"))
// 			return
// 		}

// 		next.ServeHTTP(w, r)
// 	})
// }
