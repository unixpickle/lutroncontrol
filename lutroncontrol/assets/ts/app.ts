interface Window { app: App; }

class App {
    private loaderElement: HTMLElement;
    private roomsElement: HTMLElement;
    private errorMessage: HTMLElement;

    private showingRooms: RoomView[] = [];

    constructor() {
        this.loaderElement = document.getElementById('loader');
        this.roomsElement = document.getElementById('rooms');
        this.errorMessage = document.getElementById('error-message');
        this.refresh();
    }

    async refresh() {
        let devices: LutronDevice[];
        try {
            devices = await fetchDevices();
        } catch (e) {
            this.showError('' + e);
            return;
        }
        this.showDevices(devices);
    }

    showDevices(devices: LutronDevice[]) {
        let roomToDevs = new Map<string, LutronDevice[]>();
        devices.forEach((device) => {
            let room = deviceRoom(device);
            if (roomToDevs.has(room)) {
                roomToDevs.get(room).push(device);
            } else {
                roomToDevs.set(room, [device]);
            }
        });

        // TODO: do this "reactively" by preserving existing rooms.
        this.roomsElement.innerHTML = '';
        this.showingRooms = [];
        roomToDevs.forEach((value, key) => {
            const room = new RoomView(key, value);
            this.roomsElement.appendChild(room.element);
            this.showingRooms.push(room);
        });

        document.body.className = 'status-rooms';
    }

    showError(err: string) {
        document.body.className = 'status-error';
        this.errorMessage.textContent = err;
    }
}

class View {
    private _element: HTMLElement;
    public get element(): HTMLElement {
        return this._element;
    }

    protected constructor(element: HTMLElement) {
        this._element = element;
    }
}

class RoomView extends View {
    private _name: string;
    private devices: LutronDevice[];
    private switch: RoomSwitch;

    public get name(): string {
        return this._name;
    }

    constructor(name: string, devices: LutronDevice[]) {
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

    public updateDevices(devices: LutronDevice[]) {
        this.devices = devices;
        this.switch.setLevel(this.level());
    }

    private level(): number {
        var sum = 0;
        this.devices.forEach((x) => sum += x.Level ?? 0);
        return sum / this.devices.length;
    }
}

class RoomSwitch extends View {
    public onClick: () => void = () => null;

    constructor(level: number) {
        super(document.createElement('button'));
        this.element.className = "room-switch";
        if (level == 0) {
            this.element.classList.add('room-switch-off');
        } else if (level == 100) {
            this.element.classList.add('room-switch-on');
        } else {
            this.element.classList.add('room-switch-middle');
        }
        this.element.addEventListener('click', () => this.onClick);
    }

    setLevel(level: number) {
        this.element.className = 'room-switch';
        if (level == 0) {
            this.element.classList.add('room-switch-off');
        } else if (level == 100) {
            this.element.classList.add('room-switch-on');
        } else {
            this.element.classList.add('room-switch-middle');
        }
    }
}

window.app = new App();