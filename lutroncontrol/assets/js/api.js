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
        return fetchAPI('devices');
    });
}
function fetchScenes() {
    return __awaiter(this, void 0, void 0, function* () {
        return fetchAPI('scenes');
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
function deviceName(device) {
    if (device.FullyQualifiedName.length === 0) {
        return "Unnamed Device";
    }
    return device.FullyQualifiedName[device.FullyQualifiedName.length - 1];
}
function hrefId(href) {
    if (!href) {
        return null;
    }
    const parts = href.split('/').filter((part) => part.length > 0);
    if (parts.length === 0) {
        return null;
    }
    return parts[parts.length - 1];
}
function setLevel(zoneHref, level, commandType) {
    return __awaiter(this, void 0, void 0, function* () {
        const zoneId = hrefId(zoneHref);
        if (!zoneId) {
            throw new RemoteError("invalid zone reference");
        }
        const url = `command/set_level?type=${encodeURIComponent(commandType)}` +
            `&zone=${encodeURIComponent(zoneId)}` +
            `&level=${encodeURIComponent(level.toString())}`;
        return fetchAPI(url);
    });
}
function sendZoneCommand(zoneHref, commandType) {
    return __awaiter(this, void 0, void 0, function* () {
        const zoneId = hrefId(zoneHref);
        if (!zoneId) {
            throw new RemoteError("invalid zone reference");
        }
        const url = `command/set_level?type=${encodeURIComponent(commandType)}` +
            `&zone=${encodeURIComponent(zoneId)}`;
        return fetchAPI(url);
    });
}
function pressAndRelease(buttonNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        const url = `command/press_and_release?button=${encodeURIComponent(buttonNumber.toString())}`;
        return fetchAPI(url);
    });
}
function activateScene(sceneHref) {
    return __awaiter(this, void 0, void 0, function* () {
        const sceneId = hrefId(sceneHref);
        if (!sceneId) {
            throw new RemoteError("invalid scene reference");
        }
        const url = `scene/activate?scene=${encodeURIComponent(sceneId)}`;
        return fetchAPI(url);
    });
}
function allOff() {
    return __awaiter(this, void 0, void 0, function* () {
        return fetchAPI('command/all_off');
    });
}
//# sourceMappingURL=api.js.map