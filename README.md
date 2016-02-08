# ttnmsgmap

ttnmsgmap shows gateways and messages from theThingsNetwork (demo network) on a map.

You can see this in action at:

	https://ttnmsgmap.irias.nl/

You'll see a map with Lorawan gateways. All messages on theThingsNetwork
will be displayed in a popup near the gateway that received the
packet.

The Go code in this repository is a web server with the following functionality:

- Serve the HTML and JavaScript that make up the app.
- Connect to theThingsNetwork MQTT server, and subscribe to the endpoints for packets and gateways.
- Provide server-sent event HTTP endpoints that forward the messages from the MQTT to browsers.
- Forward (and cache) requests to the theThingsNetwork gateway list (it is slow and doesn't do HTTPS).

The HTML/JavaScript app is created with:

- jQuery, Lodash
- Bootstrap
- OpenLayers with the Stamen tiles.


# compiling

	go build && sh -c '(rm assets.zip; cd assets && zip -r0 ../assets.zip .) && cat assets.zip >>ttnmsgmap'

# todo

feel free to tackle any of the todo's below.  contact me at mechiel@ueber.net.

- button to toggle showing popups for messages
- button to toggle moving the map to show popups (if you are zoomed in at a small country or city, your map will move all over the place).
- make gateways clickable, show meta info about them.
	- name?
	- list of last 10 messages received by this gateway.
	- image of device, based on eui.

- show activity of gateway by using colors
- set cors headers on the gateways api endpoint, maybe the SSE-ones too.
- find a way to workaround internet explorer not supporting server-sent-events.
- maybe have a way to list last 10 messages for a devaddr.
