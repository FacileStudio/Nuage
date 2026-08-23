package users

import documentation "github.com/FacileStudio/Nuage/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "users",
	Description: "User listing plus current-user retrieval and update routes.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/users",
			Summary:      "List users",
			Description:  "Returns all authenticated users with profile metadata.",
			Auth:         "bearer token required",
			ResponseBody: "ListResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/users/me",
			Summary:      "Return the current user",
			Description:  "Returns the authenticated user with profile metadata.",
			Auth:         "bearer token required",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "PATCH",
			Path:         "/users/me",
			Summary:      "Update the current user",
			Description:  "Updates the authenticated user's name, email, password, and/or pastel color. Replacing a password that already exists requires current_password beside password; without it the request is refused. Sending password alone gives a first password to an account that has none. Changing the email requires current_password too. A successful password change ends the account's other logins, keeps named API tokens, and rotates this session — the replacement cookie is set and the same token comes back as token for a client holding a bearer.",
			Auth:         "bearer token required",
			RequestBody:  "UpdateRequest",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body, invalid update input, a password below the length floor, a missing current_password on an account that has one, or no password to change."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid, or current_password is wrong."},
				{Status: 404, Code: "not_found", Description: "The authenticated user no longer exists."},
				{Status: 409, Code: "already_exists", Description: "A user with the same email already exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/users/me/avatar",
			Summary:      "Upload the current user's avatar",
			Description:  "Stores a new avatar file for the authenticated user and returns the updated profile. Uploading is only available to users whose identity provider supplies no photo; when it does, that photo is the one shown and this endpoint is refused.",
			Auth:         "bearer token required",
			RequestBody:  "multipart/form-data with avatar file",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Missing file, unsupported image type, or the photo is managed by single sign-on."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 404, Code: "not_found", Description: "The authenticated user no longer exists."},
				{Status: 413, Code: "resource_exhausted", Description: "Avatar file is too large."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
