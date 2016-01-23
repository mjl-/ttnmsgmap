# ttnmsgmap

ttnmsgmap shows gateways and messages from theThingsNetwork on a map.

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

- button to stop showing popups for messages
- make gateways clickable, show meta info about them.
	- name?
	- image of device, based on eui.

- show activity of gateway by using colors
- set cors headers
