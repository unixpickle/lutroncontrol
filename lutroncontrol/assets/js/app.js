var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
class App {
    constructor() {
        this.showingRooms = [];
        this.loaderElement = document.getElementById('loader');
        this.roomsElement = document.getElementById('rooms');
        this.errorMessage = document.getElementById('error-message');
        this.refresh();
    }
    refresh() {
        return __awaiter(this, void 0, void 0, function* () {
            let devices;
            try {
                devices = yield fetchDevices();
            }
            catch (e) {
                this.showError('' + e);
                return;
            }
            this.showDevices(devices);
        });
    }
    showDevices(devices) {
        let roomToDevs = new Map();
        devices.forEach((device) => {
            let room = deviceRoom(device);
            if (roomToDevs.has(room)) {
                roomToDevs.get(room).push(device);
            }
            else {
                roomToDevs.set(room, [device]);
            }
        });
        this.roomsElement.innerHTML = '';
        this.showingRooms = [];
        roomToDevs.forEach((value, key) => {
            const room = new RoomView(key, value);
            this.roomsElement.appendChild(room.element);
            this.showingRooms.push(room);
        });
        document.body.className = 'status-rooms';
    }
    showError(err) {
        document.body.className = 'status-error';
        this.errorMessage.textContent = err;
    }
}
class View {
    get element() {
        return this._element;
    }
    constructor(element) {
        this._element = element;
    }
}
class RoomView extends View {
    get name() {
        return this._name;
    }
    constructor(name, devices) {
        super(document.createElement('div'));
        this.element.className = "room-view room-view-closed";
        this._name = name;
        this.devices = devices;
        const label = document.createElement('label');
        label.className = 'room-view-name';
        label.textContent = name;
        this.element.appendChild(label);
        this.switch = new RoomSwitch(this.level());
        this.element.appendChild(this.switch.element);
    }
    updateDevices(devices) {
        this.devices = devices;
        this.switch.setLevel(this.level());
    }
    level() {
        var sum = 0;
        this.devices.forEach((x) => { var _a; return sum += (_a = x.Level) !== null && _a !== void 0 ? _a : 0; });
        return sum / this.devices.length;
    }
}
class RoomSwitch extends View {
    constructor(level) {
        super(document.createElement('button'));
        this.onClick = () => null;
        this.element.className = "room-switch";
        if (level == 0) {
            this.element.classList.add('room-switch-off');
        }
        else if (level == 100) {
            this.element.classList.add('room-switch-on');
        }
        else {
            this.element.classList.add('room-switch-middle');
        }
        this.element.addEventListener('click', () => this.onClick);
    }
    setLevel(level) {
        this.element.className = 'room-switch';
        if (level == 0) {
            this.element.classList.add('room-switch-off');
        }
        else if (level == 100) {
            this.element.classList.add('room-switch-on');
        }
        else {
            this.element.classList.add('room-switch-middle');
        }
    }
}
window.app = new App();
//# sourceMappingURL=app.js.map