package main

import (
	"errors"
	"fmt"
	"github.com/shynggys9219/greenlight/internal/data"
	"github.com/shynggys9219/greenlight/internal/validator"
	"net/http"
)

// Add a createMovieHandler for the "POST /v1/movies" endpoint.
// return a JSON response.
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	//Declare an anonymous struct to hold the information that we expect to be in the
	// HTTP request body (note that the field names and types in the struct are a subset
	// of the Movie struct that we created earlier). This struct will be our *target
	// decode destination*.
	var input struct {
		Title   string   `json:"title"`
		Year    int32    `json:"year"`
		Runtime int32    `json:"runtime"`
		Genres  []string `json:"genres"`
	}

	// if there is error with decoding, we are sending corresponding message
	err := app.readJSON(w, r, &input) //non-nil pointer as the target decode destination
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
	}

	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	err = app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": movie}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	// // Dump the contents of the input struct in a HTTP response.
	// fmt.Fprintf(w, "%+v\n", input) //+v here is adding the field name of a value // https://pkg.go.dev/fmt
}
func (app *application) createDirectorHandler(w http.ResponseWriter, r *http.Request) {
	//Declare an anonymous struct to hold the information that we expect to be in the
	// HTTP request body (note that the field names and types in the struct are a subset
	// of the Movie struct that we created earlier). This struct will be our *target
	// decode destination*.

	var input struct {
		Name    string   `json:"name"`
		Surname string   `json:"surname"`
		Awards  []string `json:"awards,omitempty"`
	}

	// if there is error with decoding, we are sending corresponding message
	err := app.readJSON(w, r, &input) //non-nil pointer as the target decode destination
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
	}

	director := &data.Director{
		Name:    input.Name,
		Surname: input.Surname,
		Awards:  input.Awards,
	}

	err = app.models.Directors.Insert(director)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/directors/%d", director.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"director": director}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	// // Dump the contents of the input struct in a HTTP response.
	// fmt.Fprintf(w, "%+v\n", input) //+v here is adding the field name of a value // https://pkg.go.dev/fmt
}

// Add a showMovieHandler for the "GET /v1/movies/:id" endpoint.
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Call the Get() method to fetch the data for a specific movie. We also need to
	// use the errors.Is() function to check if it returns a data.ErrRecordNotFound
	// error, in which case we send a 404 Not Found response to the client.
	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Title   *string  `json:"title"`
		Year    *int32   `json:"year"`
		Runtime *int32   `json:"runtime"`
		Genres  []string `json:"genres"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}
	// We also do the same for the other fields in the input struct.
	if input.Year != nil {
		movie.Year = *input.Year
	}
	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}
	if input.Genres != nil {
		movie.Genres = input.Genres // Note that we don't need to dereference a slice.
	}

	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Movies.Update(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "movie successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listMoviesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Title = app.readString(qs, "title", "")
	input.Genres = app.readCSV(qs, "genres", []string{})
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Call the GetAll() method to retrieve the movies, passing in the various filter
	// parameters.
	movies, err := app.models.Movies.GetAll(input.Title, input.Genres, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Send a JSON response containing the movie data.
	err = app.writeJSON(w, http.StatusOK, envelope{"movies": movies}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) searchByName(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name   string
		Awards []string
		data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Name = app.readString(qs, "name", "")
	input.Awards = app.readCSV(qs, "awards", []string{})
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "name", "awards", "-id", "-name", "-awards"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Call the GetAll() method to retrieve the movies, passing in the various filter
	// parameters.
	directors, err := app.models.Directors.GetAll(input.Name, input.Awards, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Send a JSON response containing the movie data.
	err = app.writeJSON(w, http.StatusOK, envelope{"directors": directors}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
