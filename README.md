# api2go

A possibly maintained [JSON API](http://jsonapi.org) Implementation for Go. Fork of the original [manyminds/api2go](https://github.com/manyminds/api2go).

## TOC
- [Installation](#installation)
- [Basic functionality](#basic-functionality)
- [Examples](#examples)
- [Interfaces to implement](#interfaces-to-implement)
  - [Responder](#responder)
  - [EntityNamer](#entitynamer)
  - [MarshalIdentifier](#marshalidentifier)
  - [UnmarshalIdentifier](#unmarshalidentifier)
  - [Marshalling with References to other structs](#marshalling-with-references-to-other-structs)
  - [Unmarshalling with references to other structs](#unmarshalling-with-references-to-other-structs)
- [Manual marshalling / unmarshalling](#manual-marshalling--unmarshalling)
- [SQL Null-Types](#sql-null-types)
- [Using api2go with the gin framework](#using-api2go-with-the-gin-framework)
- [Building a REST API](#building-a-rest-api)
  - [Query Params](#query-params)
  - [Using Pagination](#using-pagination)
  - [Fetching related IDs](#fetching-related-ids)
  - [Fetching related resources](#fetching-related-resources)
  - [Using middleware](#using-middleware)
  - [Dynamic URL Handling](#dynamic-url-handling)
- [Tests](#tests)

# Installation

For the complete api2go package use:
```go
go get github.com/jtumidanski/api2go
```

If you only need marshalling and/or unmarshalling:
```
go get github.com/jtumidanski/api2go/jsonapi 
```

## Basic functionality
Api2go will Marshal/Unmarshal exactly like the internal `json` package from Go
with one addition: It will decorate the Marshalled json with jsonapi meta
objects. Jsonapi wraps the payload inside an `attributes` object. The rest is
just Meta-Data which will be generated by api2go.

So let's take this basic example:

```go
type Article struct {
	ID    string
	Title string `json:"title"`
}
```

Would `json.Marshal` into this Json:

```json
{
  "ID": "Some-ID",
  "title": "the title"
}
```

For api2go, you have to ignore tag the `ID` field and then the result could be
something like this:

```json
{
  "type": "articles",
  "id": "1",
  "attributes": {
    "title": "Rails is Omakase"
  },
  "relationships": {
    "author": {
      "links": {
        "self": "/articles/1/relationships/author",
        "related": "/articles/1/author"
      },
      "data": { "type": "people", "id": "9" }
    }
  }
}
```

All the additional information is retrieved by implementing some interfaces.

## Examples

- Basic Examples can be found [here](https://github.com/jtumidanski/api2go/blob/master/examples/crud_example.go).
- For a more real life example implementation of api2go using [jinzhu/gorm](https://github.com/jinzhu/gorm) and [gin-gonic/gin](https://github.com/gin-gonic/gin) you can have a look at hnakamur's [repository](https://github.com/hnakamur/api2go-gorm-gin-crud-example)

## Interfaces to implement
For the following query and result examples, imagine the following 2 structs which represent a posts and
comments that belong with a has-many relation to the post.

```go
type Post struct {
  ID          int       `json:"-"`  // Ignore ID field because the ID is fetched via the
                                    // GetID() method and must not be inside the attributes object.
  Title       string    `json:"title"`
  Comments    []Comment `json:"-"` // this will be ignored by the api2go marshaller
  CommentsIDs []int     `json:"-"` // it's only useful for our internal relationship handling
}

type Comment struct {
  ID   int    `json:"-"`
  Text string `json:"text"`
}
```

You must at least implement the [MarshalIdentifier](#marshalidentifier) interface, which is the one for marshalling/unmarshalling the primary `ID` of the struct
that you want to marshal/unmarshal. This is because of the huge variety of types that you could  use for the primary ID. For example a string,
a UUID or a BSON Object for MongoDB etc...

In the Post example struct, the `ID` field is ignored because api2go will use the `GetID` method that you implemented 
for your struct to fetch the ID of the struct.
Every field inside a struct will be marshalled into the `attributes` object in
the json. In our example, we just want to have the `Title` field there.

Don't forget to name all your fields with the `json:"yourName"` tag. 

### Responder
```go
type Responder interface {
	Metadata() map[string]interface{}
	Result() interface{}
	StatusCode() int
}
```

The Responder interface must be implemented if you are using our API. It
contains everything that is needed for a response. You can see an example usage
of it in our example project.

### EntityNamer
```go
type EntityNamer interface {
	GetName() string
}
```

EntityNamer is an optional interface. Normally, the name of
a struct will be automatically generated in its plural form. For example if
your struct has the type `Post`, its generated name is `posts`. And the url
for the GET request for post with ID 1 would be `/posts/1`.

If you implement the `GetName()` method and it returns `special-posts`, then
this would be the name in the `type` field of the generated json and also the
name for the generated routes.

Currently, you must implement this interface, if you have a struct type that
consists of multiple words and you want to use a **hyphenized** name. For example `UnicornPost`.
Our default Jsonifier would then generate the name `unicornPosts`. But if you
want the [recommended](http://jsonapi.org/recommendations/#naming) name, you
have to implement `GetName`

```go
func (s UnicornPost) GetName() string {
	return "unicorn-posts"
}
```

### MarshalIdentifier
```go
type MarshalIdentifier interface {
	GetID() string
}
```

Implement this interface to marshal a struct.

### UnmarshalIdentifier
```go
type UnmarshalIdentifier interface {
	SetID(string) error
}
```

This is the corresponding interface to MarshalIdentifier. Implement this interface in order to unmarshal incoming json into
a struct.

### Marshalling with References to other structs
For relationships to work, there are 3 Interfaces that you can use:

```go
type MarshalReferences interface {
	GetReferences() []Reference
}

// MarshalLinkedRelations must be implemented if there are references and the reference IDs should be included
type MarshalLinkedRelations interface {
	MarshalReferences
	MarshalIdentifier
	GetReferencedIDs() []ReferenceID
}

// MarshalIncludedRelations must be implemented if referenced structs should be included
type MarshalIncludedRelations interface {
	MarshalReferences
	MarshalIdentifier
	GetReferencedStructs() []MarshalIdentifier
}
```

Implementing those interfaces is not mandatory and depends on your use cases. If your API has any relationships, 
you must at least implement `MarshalReferences` and `MarshalLinkedRelations`.

`MarshalReferences` must be implemented in order for api2go to know which relations are possible for your struct.

`MarshalLinkedRelations` must be implemented to retrieve the `IDs` of the relations that are connected to this struct. This method
could also return an empty array, if there are currently no relations. This is why there is the `MarshalReferences` interface, so that api2go
knows what is possible, even if nothing is referenced at the time.

In addition to that, you can implement `MarshalIncludedRelations` which exports the complete referenced structs and embeds them in the json
result inside the `included` object.

**That way you can choose how you internally manage relations.** So, there are no limits regarding the use of ORMs.

### Unmarshalling with references to other structs
Incoming jsons can also contain reference IDs. In order to unmarshal them correctly, you have to implement the following interfaces. If you only have to-one
relationships, the `UnmarshalToOneRelations` interface is enough. 

```go
// UnmarshalToOneRelations must be implemented to unmarshal to-one relations
type UnmarshalToOneRelations interface {
	SetToOneReferenceID(name, ID string) error
}

// UnmarshalToManyRelations must be implemented to unmarshal to-many relations
type UnmarshalToManyRelations interface {
	SetToManyReferenceIDs(name string, IDs []string) error
}

// UnmarshalIncludedRelations must be implemented to unmarshal included reference structs
type UnmarshalIncludedRelations interface {
    MarshalIdentifier
    SetReferencedStructs(references []Data) error
}
```

**If you need to know more about how to use the interfaces, look at our tests or at the example project.**

## Manual marshalling / unmarshalling
Please keep in mind that this only works if you implemented the previously mentioned interfaces. Manual marshalling and
unmarshalling makes sense, if you do not want to use our API that automatically generates all the necessary routes for you. You
can directly use our sub-package `github.com/jtumidanski/api2go/jsonapi` 

```go
comment1 = Comment{ID: 1, Text: "First!"}
comment2 = Comment{ID: 2, Text: "Second!"}
post = Post{ID: 1, Title: "Foobar", Comments: []Comment{comment1, comment2}}

json, err := jsonapi.Marshal(post)
```

will yield

```json
{
  "data": [
    {
      "id": "1",
      "type": "posts",
      "attributes": {
        "title": "Foobar"
      },
      "relationships": {
        "comments": {
          "data": [
            {
              "id": "1",
              "type": "comments"
            },
            {
              "id": "2",
              "type": "comments"
            }
          ]
        }
      }
    }
  ],
  "included": [
    {
      "id": "1",
      "type": "comments",
      "attributes": {
        "text": "First!"
      }
    },
    {
      "id": "2",
      "type": "comments",
      "attributes": {
        "text": "Second!"
      }
    }
  ]
}
```

You can also use `jsonapi.MarshalWithURLs` to automatically generate URLs for the rest endpoints that have a
version and BaseURL prefix. This will generate the same routes that our API uses. This adds `self` and `related` fields
for relations inside the `relationships` object.

Recover the structure from above using. Keep in mind that Unmarshalling with
included structs does not work yet. So Api2go cannot be used as a client yet.

```go
var posts []Post
err := jsonapi.Unmarshal(json, &posts)
// posts[0] == Post{ID: 1, Title: "Foobar", CommentsIDs: []int{1, 2}}
```
## SQL Null-Types
When using a SQL Database it is most likely you want to use the special SQL-Types from the `database/sql` package. These are

- sql.NullBool
- sql.NullFloat64
- sql.NullInt64
- sql.NullString

The Problem is, that they internally manage the `null` value behavior by using a custom struct. In order to Marshal and Unmarshal
these values, it is required to implement the `json.Marshaller` and `json.Unmarshaller` interfaces of the go standard library.

But you dont have to do this by yourself! There already is a library that did the work for you. We recommend that you use the types
of this library: http://gopkg.in/guregu/null.v3/zero

In order to use omitempty with those types, you need to specify them as pointers in your struct.

## Using api2go with the gin framework

If you want to use api2go with [gin](https://github.com/gin-gonic/gin) you need to use a different router than the default one.
Get the according adapter using:

```go get -tags=gingonic github.com/jtumidanski/api2go```

Currently the supported tags are: `gingonic`,`gorillamux`, or `echo`.

After that you can bootstrap api2go the following way:
```go
  import (
    "github.com/gin-gonic/gin"
    "github.com/jtumidanski/api2go"
    "github.com/jtumidanski/api2go/routing"
    "github.com/jtumidanski/api2go/examples/model"
    "github.com/jtumidanski/api2go/examples/resource"
    "github.com/jtumidanski/api2go/examples/storage"
  )

  func main() {
    r := gin.Default()
    api := api2go.NewAPIWithRouting(
      "api",
      api2go.NewStaticResolver("/"),
      routing.Gin(r),
    )

    userStorage := storage.NewUserStorage()
    chocStorage := storage.NewChocolateStorage()
    api.AddResource(model.User{}, resource.UserResource{ChocStorage: chocStorage, UserStorage: userStorage})
    api.AddResource(model.Chocolate{}, resource.ChocolateResource{ChocStorage: chocStorage, UserStorage: userStorage})

    r.GET("/ping", func(c *gin.Context) {
      c.String(200, "pong")
    })
    r.Run(":8080")
  }
```

Keep in mind that you absolutely should map api2go under its own namespace to not get conflicts with your normal routes.

If you need api2go with any different go framework, just send a PR with the according adapter :-)

## Building a REST API

First, write an implementation of either `api2go.ResourceGetter`, `api2go.ResourceCreator`, `api2go.ResourceUpdater`,  `api2go.ResourceDeleter`, or any combination of them.
You can also write an implementation the `CRUD` interface which embed all of them.
You have to implement at least one of these 4 methods:

```go
type fixtureSource struct {}

// FindOne returns an object by its ID
// Possible success status code 200
func (s *fixtureSource) FindOne(ID string, r api2go.Request) (Responder, error) {}

// Create a new object. Newly created object/struct must be in Responder.
// Possible status codes are:
// - 201 Created: Resource was created and needs to be returned
// - 202 Accepted: Processing is delayed, return nothing
// - 204 No Content: Resource created with a client generated ID, and no fields were modified by
//   the server
func (s *fixtureSource) Create(obj interface{}, r api2go.Request) (Responder, err error) {}

// Delete an object
// Possible status codes are:
// - 200 OK: Deletion was a success, returns meta information, currently not implemented! Do not use this
// - 202 Accepted: Processing is delayed, return nothing
// - 204 No Content: Deletion was successful, return nothing
func (s *fixtureSource) Delete(id string, r api2go.Request) (Responder, err error) {}

// Update an object
// Possible status codes are:
// - 200 OK: Update successful, however some field(s) were changed, returns updates source
// - 202 Accepted: Processing is delayed, return nothing
// - 204 No Content: Update was successful, no fields were changed by the server, return nothing
func (s *fixtureSource) Update(obj interface{}, r api2go.Request) (Responder, err error) {}
```

If you want to return a jsonapi compatible error because something went wrong inside the CRUD methods, you can use our
`HTTPError` struct, which can be created with `NewHTTPError`. This allows you to set the error status code and add
as many information about the error as you like. See: [jsonapi error](http://jsonapi.org/format/#errors)

To fetch all objects of a specific resource you can choose to implement one or both of the following
interfaces:

```go
type FindAll interface {
	// FindAll returns all objects
	FindAll(req Request) (Responder, error)
}

type PaginatedFindAll interface {
	PaginatedFindAll(req Request) (totalCount uint, response Responder, err error)
}
```

`FindAll` returns everything. You could limit the results only by using Query Params which are described [here](#query-params)

`PaginatedFindAll` can also use Query Params, but in addition to that it does not need to send all objects at once and can split
up the result with pagination. You have to return the total number of found objects in order to let our API automatically generate
pagination links. More about pagination is described [here](#using-pagination)

You can then create an API:

```go
api := api2go.NewAPI("v1")
api.AddResource(Post{}, &PostsSource{})
http.ListenAndServe(":8080", api.Handler())
```

Instead of `api2go.NewAPI` you can also use `api2go.NewAPIWithBaseURL("v1", "http://yourdomain.com")` to prefix all
automatically generated routes with your domain and protocoll.

This generates the standard endpoints:

```
OPTIONS /v1/posts
OPTIONS /v1/posts/<id>
GET     /v1/posts
POST    /v1/posts
GET     /v1/posts/<id>
PATCH   /v1/posts/<id>
DELETE  /v1/posts/<id>
GET     /v1/posts/<id>/comments            // fetch referenced comments of a post
GET     /v1/posts/<id>/relationships/comments      // fetch IDs of the referenced comments only
PATCH   /v1/posts/<id>/relationships/comments      // replace all related comments

// These 2 routes are only created for to-many relations that implement EditToManyRelations interface
POST    /v1/posts/<id>/relationships/comments      // Add a new comment reference, only for to-many relations
DELETE  /v1/posts/<id>/relationships/comments      // Delete a comment reference, only for to-many relations
```

For the last two generated routes, it is necessary to implement the `jsonapi.EditToManyRelations` interface.

```go
type EditToManyRelations interface {
	AddToManyIDs(name string, IDs []string) error
	DeleteToManyIDs(name string, IDs []string) error
}
```

All PATCH, POST and DELETE routes do a `FindOne` and update the values/relations in the previously found struct. This
struct will then be passed on to the `Update` method of a resource struct. So you get all these routes "for free" and just
have to implement the `ResourceUpdater` `Update` method.

### Query Params
To support all the features mentioned in the `Fetching Resources` section of Jsonapi:
http://jsonapi.org/format/#fetching

If you want to support any parameters mentioned there, you can access them in your Resource
via the `api2go.Request` Parameter. This currently supports `QueryParams` which holds
all query parameters as `map[string][]string` unfiltered. So you can use it for:
  * Filtering
  * Inclusion of Linked Resources
  * Sparse Fieldsets
  * Sorting
  * Aything else you want to do that is not in the official Jsonapi Spec

```go
type fixtureSource struct {}

func (s *fixtureSource) FindAll(req api2go.Request) (Responder, error) {
  for key, values range req.QueryParams {
    ...
  }
  ...
}
```

If there are multiple values, you have to separate them with a comma. api2go automatically
slices the values for you.

```
Example Request
GET /people?fields=id,name,age

req.QueryParams["fields"] contains values: ["id", "name", "age"]
```

### Using Pagination
Api2go can automatically generate the required links for pagination. Currently there are 2 combinations of query
parameters supported:

- page[number], page[size]
- page[offset], page[limit]

Pagination is optional. If you want to support pagination, you have to implement the `PaginatedFindAll` method
in you resource struct. For an example, you best look into our example project.

Example request

```
GET /v0/users?page[number]=2&page[size]=2
```

would return a json with the top level links object

```json
{
  "links": {
    "first": "http://localhost:31415/v0/users?page[number]=1&page[size]=2",
    "last": "http://localhost:31415/v0/users?page[number]=5&page[size]=2",
    "next": "http://localhost:31415/v0/users?page[number]=3&page[size]=2",
    "prev": "http://localhost:31415/v0/users?page[number]=1&page[size]=2"
  },
  "data": [...]
}
```

### Fetching related IDs
The IDs of a relationship can be fetched by following the `self` link of a relationship object in the `links` object
of a result. For the posts and comments example you could use the following generated URL:

```
GET /v1/posts/1/relationships/comments
```

This would return all comments that are currently referenced by post with ID 1. For example:

```json
{
  "links": {
    "self": "/v1/posts/1/relationships/comments",
    "related": "/v1/posts/1/comments"
  },
  "data": [
    {
      "type": "comments",
      "id": "1"
    },
    {
      "type":"comments",
      "id": "2"
    }
  ]
}
```

### Fetching related resources
Api2go always creates a `related` field for elements in the `relationships` object of the result. This is like it's
specified on jsonapi.org. Post example:

```json
{
  "data": [
    {
      "id": "1",
      "type": "posts",
      "title": "Foobar",
      "relationships": {
        "comments": {
          "links": {
            "related": "/v1/posts/1/comments",
            "self": "/v1/posts/1/relationships/comments"
          },
          "data": [
            {
              "id": "1",
              "type": "comments"
            },
            {
              "id": "2",
              "type": "comments"
            }
          ]
        }
      }
    }
  ]
}
```

If a client requests this `related` url, the `FindAll` method of the comments resource will be called with a query
parameter `postsID`.

So if you implement the `FindAll` method, do not forget to check for all possible query Parameters. This means you have
to check all your other structs and if it references the one for that you are implementing `FindAll`, check for the
query Paramter and only return comments that belong to it. In this example, return the comments for the Post.

### Using middleware
We provide a custom `APIContext` with
a [context](https://godoc.org/context) implementation that you
can use if you for example need to check if a user is properly authenticated
before a request reaches the api2go routes.

You can either use our struct or implement your own with the `APIContexter`
interface

```go
type APIContexter interface {
    context.Context
    Set(key string, value interface{})
    Get(key string) (interface{}, bool)
    Reset()
}
```

If you implemented your own `APIContexter`, don't forget to define
a `APIContextAllocatorFunc` and set it with `func (api *API) SetContextAllocator(allocator APIContextAllocatorFunc)`

But in most cases, this is not needed.

To use a middleware, it is needed to implement our
`type HandlerFunc func(APIContexter, http.ResponseWriter, *http.Request)`. A `HandlerFunc` can then be 
registered with `func (api *API) UseMiddleware(middleware ...HandlerFunc)`. You can either pass one or many middlewares 
that will be executed in order before any other api2go routes. Use this to set up database connections, user authentication
and so on.

### Dynamic URL handling
If you have different TLDs for one api, or want to use different domains in development and production, you can implement a custom
URLResolver in api2go. 

There is a simple interface, which can be used if you get TLD information from the database, the server environment, or anything else
that's not request dependant:
```go
type URLResolver interface {
	GetBaseURL() string
}
```
And a more complex one that also gets request information:
```go
type RequestAwareURLResolver interface {
	URLResolver
	SetRequest(http.Request)
}
```

For most use cases we provide a CallbackResolver which works on a per request basis and may fill
your basic needs. This is particulary useful if you are using an nginx proxy which sets `X-Forwarded-For` headers.

```go
resolver := NewCallbackResolver(func(r http.Request) string{})
api := NewApiWithMarshalling("v1", resolver, marshalers)
```
## Tests

```sh
go test ./...
ginkgo -r                # Alternative
ginkgo watch -r -notify  # Watch for changes
```
