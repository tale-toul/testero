## TODO List

* Variable HIGHMEMLIM needs to be defined only at the beggining of the program execution, similar to how HIGHFILELIM is dealt with (DONE)

* Variables: HIGHMEMLIM and HIGHFILELIM are never updated once defined at the beginning of the program: If they are set to default values they should be updated after every add/remove request; if they are set from environment variables they should be updated too, but using a different mechanism.

* Existing files should be deleted when application terminates or is interrupted, to avoid filling disk with old files (DONE)

* Creation of data for memory parts and files should be redisigned to reduce CPU usage. 

* Add functionality to request CPU load for a certain time in seconds

* Create a simple web interface to call the API endpoints
