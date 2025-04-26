var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
class RemoteError extends Error {
    constructor(msg) {
        super(msg);
        Object.setPrototypeOf(this, RemoteError.prototype);
    }
}
function fetchDevices() {
    return __awaiter(this, void 0, void 0, function* () {
        return fetchAPI('/devices');
    });
}
function fetchAPI(url) {
    return __awaiter(this, void 0, void 0, function* () {
        const obj = yield (yield fetch(url)).json();
        if (obj.hasOwnProperty("error")) {
            throw new RemoteError(obj["error"]);
        }
        return obj;
    });
}
function deviceRoom(device) {
    if (device.FullyQualifiedName.length === 1) {
        return "Other";
    }
    else {
        return device.FullyQualifiedName[0];
    }
}
//# sourceMappingURL=api.js.map