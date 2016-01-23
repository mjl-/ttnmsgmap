'use strict';

$(function() {

var filters = {
	devAddr: '',
	gatewayEui: ''
};

var gatewaySource = new ol.source.Vector({});
window.gatewaySource = gatewaySource;

var packetSource = new ol.source.Vector({});

var gatewayLayer = new ol.layer.Vector({
	source: gatewaySource,
	style: new ol.style.Style({
		image: new ol.style.Circle({
			radius: 8,
			fill: new ol.style.Fill({
				color: '#FF0051'
			}),
			stroke: new ol.style.Stroke({
				color: 'white',
				width: 2
			})
		})
	})
});
var packetLayer = new ol.layer.Vector({
	source: packetSource
});

var container = document.getElementById('popup');
var content = document.getElementById('popup-content');
var closer = document.getElementById('popup-closer');

closer.onclick = function() {
	overlay.setPosition(undefined);
	closer.blur();
	return false;
};

var overlay = new ol.Overlay({
	element: container,
	autoPan: true,
	autoPanAnimation: {
		duration: 250
	}
});

var map = new ol.Map({
	target: 'map',
	overlays: [overlay],
	layers: [
		new ol.layer.Tile({
			source: new ol.source.Stamen({
				layer: 'watercolor'
			})
		}),
		gatewayLayer,
		packetLayer

		/*
		new ol.layer.Tile({
			source: new ol.source.Stamen({
				layer: 'terrain-labels'
			})
		}) */
	],
	view: new ol.View({
		center: ol.proj.transform([0, 0], 'EPSG:4326', 'EPSG:3857'),
		zoom: 2
	})
});
window.map = map;

$(window).resize(function() {
	$('#map').height($(window).height());
	map.updateSize();
}).resize();


if(!window.EventSource) {
	alert('Server-sent events are not supported by your browser.');
	return;
}
	
var gatewaySrc = new EventSource('sub/gateways/');
var packetSrc = new EventSource('sub/packets/');

var gatewayMap = {};
var gatewayCount = 0;
var packetCount = 0;
var eventCount = 0;
window.gatewayMap = gatewayMap;

var wgs2webmercator = function(coord) {
	return ol.proj.transform(coord, 'EPSG:4326', 'EPSG:3857');
};

var $status = $('.x-status');
var updateStatus = function() {
	$status.text(''+gatewayCount+' gateways, '+packetCount+' packets, '+eventCount+' events');
};

var addGateway = function(gw) {
	var isNew = !gatewayMap[gw.eui];
	gatewayMap[gw.eui] = gw;
	if(isNew) {
		gatewayCount += 1;
		updateStatus();
		if(gw.eui.indexOf(filters.gatewayEui) !== 0) {
			return;
		}
		var coord = wgs2webmercator([gw.longitude, gw.latitude]);
		var f = new ol.Feature({
			geometry: new ol.geom.Point(coord)
		});
		gatewaySource.addFeature(f);
	}
};

var redrawGateways = function() {
	gatewaySource.clear();
	_.each(gatewayMap, function(gw) {
		if(gw.eui.indexOf(filters.gatewayEui) !== 0) {
			return;
		}
		var coord = wgs2webmercator([gw.longitude, gw.latitude]);
		var f = new ol.Feature({
			geometry: new ol.geom.Point(coord)
		});
		gatewaySource.addFeature(f);
	});
};

gatewaySrc.addEventListener('message', function(e) {
	var gw = JSON.parse(e.data);
	addGateway(gw);
	eventCount += 1;
	updateStatus();
}, false);

$.getJSON('/gateways/', function(gateways) {
	var delay = 0;
	_.each(gateways, function(gw) {
		setTimeout(function() {
			addGateway(gw);
		}, delay);
		delay += 30;
	});
});

var messageQueue = [];

setInterval(function() {
	while(messageQueue.length > 0) {
		var msg = messageQueue.shift();

		var gw = msg.gateway;
		var p = msg.packet;

		if(false && (!gw.longitude || !gw.latitude)) {
			continue;
		}

		if(gw.eui.indexOf(filters.gatewayEui) !== 0) {
			continue;
		}
		if(p.devAddr.indexOf(filters.devAddr) !== 0) {
			continue;
		}

		var coord = wgs2webmercator([gw.longitude, gw.latitude]);

		var $content = $(content);
		$content.find('.x-data').text(''+p.data);
		$content.find('.x-datahex').text(''+p.dataHex);
		$content.find('.x-devaddr').text(''+p.devAddr);
		$content.find('.x-gateway').text(''+p.gatewayEui);
		$content.find('.x-signal').text([p.dataRate, ''+p.frequency, ''+p.rssi, ''+p.snr].join(' / '));
		var secs = parseInt((new Date().getTime() - p.received)/1000);
		$content.find('.x-time').text(''+secs+' seconds ago');
		overlay.setPosition(coord);

		break;
	}
	
}, 1*1000);

packetSrc.addEventListener('message', function(e) {
	packetCount += 1;
	eventCount += 1;
	updateStatus();

	var packet = JSON.parse(e.data);

	var gw = gatewayMap[packet.gatewayEui];
	if(!gw) {
		console.log('packet for unknown gateway?!');
		return;
	}

	packet.received = new Date().getTime();
	messageQueue.unshift({packet: packet, gateway: gw});
	messageQueue = messageQueue.slice(0, 100);
}, false);


$('.x-filters').on('submit', function(e) {
	e.preventDefault();

	filters.devAddr = $('.x-filters [name=devaddr]').val().toUpperCase();
	filters.gatewayEui = $('.x-filters [name=gatewayeui]').val().toUpperCase();
	console.log('new filters', filters);
	redrawGateways();
});


});
