import {HTTPRequest, WS} from './lib/conn.js';
import {RPC} from './lib/rpc.js';
import {Arr, Bool, Int, Null, Obj, Or, Rec, Str} from './lib/typeguard.js';

const rpc = new RPC();

export const
isStr = Str(),
isStrArr = Arr(isStr),
rpcInit = (url: string) => WS(url).then(conn => rpc.reconnect(conn)),
environmentUpdate = rpc.subscribe(-1, Rec(isStr, Or(Obj({
	Tags: isStrArr,
	Packages: isStrArr,
	Description: isStr,
	ReadMe: isStr,
	Status: Int(0, 2),
	SoftPack: Bool()
}), Null()))),
getUserGroups = (user: string) => HTTPRequest("ldap", {"method": "POST", "data": user, "response": "json", "checker": isStrArr});
