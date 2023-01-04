import http from 'k6/http';
import { vu } from 'k6/execution';
import { sleep, check } from 'k6';
import { SharedArray } from 'k6/data';
import crypto from 'k6/crypto';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';


//const objectContent = "Hello World!";
const objectContent = open('object.bin', 'b');

const numObjects = 64;
const hosts = ["http://localhost:5000", "http://localhost:5001"]
const objectURLs = new SharedArray('objects', function () {
    let k6InstanceID = uuidv4(true);
    let objects = new Array();
    for(let i = 0; i < numObjects; i++){
        //let hex = Number(i).toString(16).padStart(64, "0");
        let hex = crypto.sha256(`${k6InstanceID}/${i}`, 'hex').toUpperCase();
        let host = hosts[parseInt(hex.substr(-5), 16) % hosts.length]; // substr -> 2^256 can not be represented without a loss of precision
        let url = `${host}/object/${hex}`;
        objects.push(url);
    }
  return objects;
});

export function setup() {
  for(let i = 0; i < numObjects; i++){
    let url = objectURLs[i];
    http.del(url);

    http.put(url, {"file": http.file(objectContent, 'file')});
  }
}

export default function () {
    let object = (vu.idInTest - 1);
    if (object >= objectURLs.length) {
      test.abort("every VU needs its own object");
    }
    let objectURL = objectURLs[object];

    let res = http.get(objectURL);
    //sleep(1);
    check(res, {
        'is status 200': (r) => r.status === 200,
    });

}

export function teardown(data) {
  for(let i = 0; i < numObjects; i++){
    let url = objectURLs[i];
    http.del(url);
  }
}
