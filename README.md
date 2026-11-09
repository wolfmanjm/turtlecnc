A conversational CAD/CAM.
Can generate GCode for simple CNC jobs by filling in a minimum of parameters.

Will provide a HTTP server or a command line TUI interface for generating the gcode.

To run in a console

	> turtlecnc -tui

A list will pop up in the terminal (console mode), select the job by using TAB
or typing the number or clicking with a mouse.

A form will popup with the parameters for the job. Some are prefilled with
defaults but maybe changed. Navigate the fields using TAB or SHIFT TAB, or
click the field with a mouse, when the value has been entered type Enter
which will go to the next field.

The Job Name field will be the name of the file generated, postpended with .nc

Navigate to the OK button and hit enter (or click OK with a mouse). The file
will be generated (Please check it carefully, air run it or run it through
camotics or similar).

At any time ESC will exit the program. When in a form typing ? or selecting
the HELP button will show information for the current form, documenting each
field.


To run the web server at port 8080 (the default)

	> turtlecnc -port 8080 -s

then connect to the server with a web browser (eg http://localhost:8080). A
similar list of jobs and forms will be displayed.

A server running this is available at https://cam.wolfman.com

Full documentation for each job is available at https://cam.wolfman.com/documentation.html
