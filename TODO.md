## CHANGELOG

* Existing files should be deleted when application terminates or is interrupted, to avoid filling disk with old files (DONE)

* Add functionality to request CPU load for a certain time in seconds (DONE)

* Add an API endpoint to stop the CPU load inmediately (DONE)

## TODO List

* Variable HIGHMEMLIM needs to be defined only at the beggining of the program execution, similar to how HIGHFILELIM is dealt with

* Variables: HIGHMEMLIM and HIGHFILELIM are never updated once defined at the beginning of the program: If they are set to default values they should be updated after every add/remove request; if they are set from environment variables they should be updated too, but using a different mechanism.

* Creation of data for memory parts and files should be redisigned to reduce CPU usage. 

* Create a simple web interface to call the API endpoints

* Add an environment variable to allow the configuration of the number to be factorized to generate CPU load

* Add environment var to define IP and PORT where the web server will listen on

* Add https support

* Require ID token to make requests

* Add API cpu endpoint to return the number to factor currently being used

* Add API cpu endpoint to return data about the current load request, if any: when was the request sent, and the time requested
