listings
========

Language Choice
---------------

listings is written in Go, which has a standard library for both http and
sql, meaning only a sql driver library was needed to make the api.

Future Improvements
-------------------

Given more time, the following would be added to the project:

1. **Tests**: both to better ensure correctness, and to make refactoring easier
2. **Pagination**: the api can potentially display a large number of results
3. **Remove Global State for Database**: the database information would be passed as
context to the http handler rather than set as a global variable
4. **Authentication**: allow for access control
5. **Better sql queries**: the current query is based on the dataset, and would
need to be improved for a different or larger dataset
