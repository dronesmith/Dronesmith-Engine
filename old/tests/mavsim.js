'use strict';

var path = require('path'),
  mavlink = require('mavlink'),
  events = require('events'),
  dgram = require('dgram'),
  fs = require('fs');

if (process.argv.length < 3) {
  console.log(
    "mavsim generates simulated mavlink data by replaying recorded MAV file over and over.\n"
    + "You will need to use mavrecorder to actually generate flight data from a real mav--this is"
    + " not a true SITL. The mavlink data is sent out as binary MAVLink packets.");
  console.log("\t--file\tSpecify a mavlink file in binary format. Defaults to a file located in tools/");
  console.log("\t--port\tPort to output to. Default is 4001.");
  console.log("\t--loop\tNumber of times to loop. Default is once.");
  console.log("\t--forever\tLoops forever. Only way out is ctrl+x,c");
  console.log("\t--frequency\tHow many times a second you want a message. Default is 60Hz.")
} else {
  var mav = new mavlink(1,1);

  mav.on('ready', function() {
    var
      filedest = path.join(path.resolve(__dirname), 'noMotorLogTest.mavlink'),
      port = 14550,
      loop = 1,
      forever = false,
      freq = 60;

    for (var k in process.argv) {
      var arg = process.argv[k];

      switch (arg) {
        case '--loop':
          loop = process.argv[+k+1] || 1;
          break;
        case '--port':
          port = +process.argv[+k+1] || 4001;
          break;
        case '--file':
          filedest = path.resolve(process.argv[+k+1]) || path.resolve(__dirname);
          break;
        case '--forever':
          forever = true;
          break;
        case '--frequency':
          freq = process.argv[+k+1] || 60;
          break;
      }
    }

    // init udp
    var socket = dgram.createSocket('udp4');

    // load file
    var data = fs.readFileSync(filedest);

    console.log("Broadcasting on", port, "...");

    var i = loop;
    var ptr = 0;

    var obj = setInterval(function() {
      if (ptr >= data.length) {
        if (forever || --i > 0) {
          console.log("Resetting flight.");
          ptr = 0;
        } else {
          console.log("Completed.");
          socket.close();
          clearInterval(obj);
          return;
        }
      }
      try {
        var date = new Date(data.readUIntBE(ptr, 8));
        ptr += 8;
        var len = data[ptr+1] + ptr + 8;
        var msgBuff = JSON.parse(JSON.stringify(data.slice(ptr, len)));
        ptr = len;
      } catch(e) {
        console.log("[Error]", e);
        socket.close();
        process.exit(-1);
      }

      var buff = Buffer(msgBuff);
      socket.send(buff, 0, buff.length, port, "localhost", function(err) {
        if (err) {
          console.log("[Error]", err);
          socket.close();
          process.exit(-1);
        }
      });
    }, Math.round((1 / freq) * 1000));

  });
}
