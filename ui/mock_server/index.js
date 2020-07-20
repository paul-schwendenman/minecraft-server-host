const express = require('express');
const cors = require("cors");

const app = express();

const allowedOrigins = ["http://localhost:5000"];

app.use(
    cors({
        origin: allowedOrigins,
    })
);

const ipv4 = {
    random: function (subnet, mask) {
      // generate random address (integer)
      // if the mask is 20, then it's an integer between
      // 1 and 2^(32-20)
      let randomIp = Math.floor(Math.random() * Math.pow(2, 32 - mask)) + 1;

      return this.lon2ip(this.ip2lon(subnet) | randomIp);
    },
    ip2lon: function (address) {
      let result = 0;

      address.split('.').forEach(function(octet) {
        result <<= 8;
        result += parseInt(octet, 10);
      });

      return result >>> 0;
    },
    lon2ip: function (lon) {
      return [lon >>> 24, lon >> 16 & 255, lon >> 8 & 255, lon & 255].join('.');
    }
  };

function sleep(ms) {
    return new Promise((resolve) => {
        setTimeout(resolve, ms);
    });
}

let errorThreshold = 0.9;
const state = {
    instance: {
        state: "stopped",
        ip_address: null
    },
    dns_record: {
        name: "minecraft.example.com.",
        value: "10.0.0.1",
        type: "A"
    }
};

const details_one_player = {
    description: {
        text: "A Minecraft Server"
    },
    players: {
        max: 20,
        online: 1,
        sample: [
        {
            id: "cdce37cd-2215-42ef-a4a4-c8b9189c9259",
            name: "example"
        }
        ]
    },
    version: {
        name: "1.15.2",
        protocol: 578
    }
};

const details_two_players = {
    description: {
        text: "A Minecraft Server"
    },
    players: {
        max: 20,
        online: 2,
        sample: [
        {
            id: "cdce37cd-2215-42ef-a4a4-c8b9189c9259",
            name: "example"
        },
        {
            id: "d720a93f-da90-41fa-8653-d09d81fa4b77",
            name: "example2"
        }
        ]
    },
    version: {
        name: "1.15.2",
        protocol: 578
    }
};

const details_zero_players = {
    description: {
      text: "A Minecraft Server"
    },
    players: {
      max: 20,
      online: 0
    },
    version: {
      name: "1.15.2",
      protocol: 578
    }
};

app.get('/start', async (req, res) => {
    state.instance.state = "pending";
    await sleep(250);

    setTimeout(() => {
        state.instance.state = "running";
        state.instance.ip_address = ipv4.random('10.0.0.0', 8);
    }, 1000)
    res.setHeader('Content-Type', 'application/json');
    res.send({"message": "Success"});
});

app.get('/stop', async (req, res) => {
    state.instance.state = "stopping";
    await sleep(500);

    setTimeout(() => {
        state.instance.state = "stopped";
        state.instance.ip_address = null;
    }, 5000)
    errorThreshold = 0.9;
    res.setHeader('Content-Type', 'application/json');
    res.send({"message": "Success"});
});

app.get('/syncdns', async (req, res) => {
    state.dns_record.value = state.instance.ip_address;
    await sleep(250);
    setTimeout(() => {
        errorThreshold = 0.2;
    }, 1000);

    res.setHeader('Content-Type', 'application/json');
    res.send({"message": "Success"});
});

app.get('/status', async (req, res) => {
    await sleep(100);
    res.setHeader('Content-Type', 'application/json');
    res.send(state);
});

app.get('/details', async (req, res) => {
    const { hostname } = req.query;
    if (!hostname) {
        res.sendStatus(400);
    }
    if (Math.random() < errorThreshold) {
        const errorCode = [
            500,
            503,
            504,
        ][Math.floor(Math.random() * 3)];

        res.sendStatus(errorCode);
        return;
    }
    await sleep(333);

    const details = [
        details_zero_players,
        details_one_player,
        details_two_players
    ][Math.floor(Math.random() * 3)];

    res.setHeader('Content-Type', 'application/json');
    res.send(details);
})

app.listen(5001, () =>
  console.log('Express server is running on localhost:5001')
);
