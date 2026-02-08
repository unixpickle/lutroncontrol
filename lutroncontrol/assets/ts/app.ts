interface Window { app: App; }

type BannerKind = 'info' | 'success' | 'error';

type BannerFn = (message: string, kind: BannerKind) => void;

type RefreshFn = () => void;

class App {
    private loaderElement: HTMLElement;
    private roomsElement: HTMLElement;
    private scenesElement: HTMLElement;
    private errorMessage: HTMLElement;
    private statusBadge: HTMLElement;
    private banner: HTMLElement;
    private refreshButton: HTMLButtonElement;
    private allOffButton: HTMLButtonElement;
    private refreshing = false;
    private refreshTimer: number | null = null;

    constructor() {
        this.loaderElement = document.getElementById('loader');
        this.scenesElement = document.getElementById('scenes');
        this.roomsElement = document.getElementById('rooms');
        this.errorMessage = document.getElementById('error-message');
        this.statusBadge = document.getElementById('status-badge');
        this.banner = document.getElementById('banner');
        this.refreshButton = document.getElementById('refresh-button') as HTMLButtonElement;
        this.allOffButton = document.getElementById('all-off-button') as HTMLButtonElement;
        this.refreshButton.addEventListener('click', () => this.refresh(true));
        this.allOffButton.addEventListener('click', async () => {
            try {
                await allOff();
                this.setStatus('All lights off', 'success');
                this.refresh(false);
            } catch (e) {
                this.setStatus('' + e, 'error');
            }
        });
        this.refresh(true);
        this.refreshTimer = window.setInterval(() => this.refresh(false), 60000);
    }

    async refresh(showLoading: boolean) {
        if (this.refreshing) {
            return;
        }
        this.refreshing = true;
        if (showLoading) {
            document.body.className = 'status-loading';
        }
        this.setStatus('Refreshing...', 'info');
        let devices: LutronDevice[];
        let scenes: SceneInfo[] = [];
        try {
            devices = await fetchDevices();
            scenes = await fetchScenes();
        } catch (e) {
            this.showError('' + e);
            this.refreshing = false;
            return;
        }
        this.showDevices(devices);
        this.showScenes(scenes);
        this.setStatus('Updated just now', 'success');
        this.refreshing = false;
    }

    showDevices(devices: LutronDevice[]) {
        const roomToDevs = new Map<string, LutronDevice[]>();
        devices.forEach((device) => {
            const room = deviceRoom(device);
            if (roomToDevs.has(room)) {
                roomToDevs.get(room).push(device);
            } else {
                roomToDevs.set(room, [device]);
            }
        });

        const entries = Array.from(roomToDevs.entries());
        entries.sort((a, b) => {
            const aIsOther = a[0] === 'Other';
            const bIsOther = b[0] === 'Other';
            if (aIsOther && !bIsOther) {
                return 1;
            }
            if (!aIsOther && bIsOther) {
                return -1;
            }
            return a[0].localeCompare(b[0]);
        });

        this.roomsElement.innerHTML = '';
        entries.forEach(([key, value]) => {
            const room = new RoomView(key, value, (message, kind) => this.setStatus(message, kind), () => this.refresh(false));
            this.roomsElement.appendChild(room.element);
        });

        document.body.className = 'status-rooms';
    }

    showScenes(scenes: SceneInfo[]) {
        this.scenesElement.innerHTML = '';
        const programmed = scenes.filter((scene) => scene.IsProgrammed);
        if (programmed.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'scene-empty';
            empty.textContent = 'No programmed scenes found.';
            this.scenesElement.appendChild(empty);
            return;
        }
        programmed.forEach((scene) => {
            const button = document.createElement('button');
            button.className = 'scene-button';
            button.textContent = scene.Name || `Scene ${scene.ButtonNumber + 1}`;
            button.addEventListener('click', async () => {
                try {
                    await activateScene(scene.href);
                    this.setStatus(`Activated ${button.textContent}`, 'success');
                } catch (e) {
                    this.setStatus('' + e, 'error');
                }
            });
            this.scenesElement.appendChild(button);
        });
    }

    showError(err: string) {
        document.body.className = 'status-error';
        this.errorMessage.textContent = err;
    }

    private setStatus(message: string, kind: BannerKind) {
        this.statusBadge.textContent = message;
        this.statusBadge.className = `status-badge status-${kind}`;
        this.banner.textContent = '';
        this.banner.className = 'banner';
        if (kind === 'error') {
            console.error(message);
        }
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
    private devicesElement: HTMLElement;
    private summaryElement: HTMLElement;
    private notify: BannerFn;
    private refresh: RefreshFn;

    public get name(): string {
        return this._name;
    }

    constructor(name: string, devices: LutronDevice[], notify: BannerFn, refresh: RefreshFn) {
        super(document.createElement('section'));
        this.element.className = "room-card";
        this._name = name;
        this.devices = devices;
        this.notify = notify;
        this.refresh = refresh;

        const header = document.createElement('div');
        header.className = 'room-header';

        const titleWrap = document.createElement('div');
        titleWrap.className = 'room-title-wrap';

        const label = document.createElement('h2');
        label.className = 'room-title';
        label.textContent = name;
        titleWrap.appendChild(label);

        this.summaryElement = document.createElement('div');
        this.summaryElement.className = 'room-summary';
        titleWrap.appendChild(this.summaryElement);

        header.appendChild(titleWrap);
        this.element.appendChild(header);

        this.devicesElement = document.createElement('div');
        this.devicesElement.className = 'device-grid';
        this.element.appendChild(this.devicesElement);

        this.renderDevices();
    }

    public updateDevices(devices: LutronDevice[]) {
        this.devices = devices;
        this.renderDevices();
    }

    private renderDevices() {
        this.devicesElement.innerHTML = '';
        this.devices.forEach((device) => {
            const deviceView = new DeviceView(device, this.notify, this.refresh);
            this.devicesElement.appendChild(deviceView.element);
        });
        const average = this.averageLevel();
        const averageText = average === null ? 'No dimmers' : `Avg ${Math.round(average)}%`;
        this.summaryElement.textContent = `${this.devices.length} device${this.devices.length === 1 ? '' : 's'} · ${averageText}`;
    }

    private averageLevel(): number | null {
        let count = 0;
        let sum = 0;
        this.devices.forEach((device) => {
            if (device.Level !== undefined) {
                sum += device.Level;
                count += 1;
            }
        });
        if (count === 0) {
            return null;
        }
        return sum / count;
    }
}

class DeviceView extends View {
    private device: LutronDevice;
    private notify: BannerFn;
    private refresh: RefreshFn;
    private levelValue?: HTMLElement;
    private slider?: HTMLInputElement;
    private busy = false;

    constructor(device: LutronDevice, notify: BannerFn, refresh: RefreshFn) {
        super(document.createElement('div'));
        this.device = device;
        this.notify = notify;
        this.refresh = refresh;
        this.element.className = 'device-card';

        const header = document.createElement('div');
        header.className = 'device-header';

        const title = document.createElement('div');
        title.className = 'device-title';
        title.textContent = deviceName(device);
        header.appendChild(title);

        const meta = document.createElement('div');
        meta.className = 'device-meta';
        meta.textContent = device.DeviceType;
        header.appendChild(meta);

        this.element.appendChild(header);

        const controls = document.createElement('div');
        controls.className = 'device-controls';
        this.element.appendChild(controls);

        if (device.Zone && device.DeviceType === 'QsWirelessShade') {
            this.buildShadeControls(controls, device);
        } else if (device.Zone) {
            if (device.Level !== undefined) {
                this.buildDimmerControls(controls, device);
            } else {
                this.buildSwitchControls(controls, device);
            }
        }

        if (device.Buttons && device.Buttons.length > 0) {
            this.buildButtonControls(device.Buttons);
        }
    }

    private buildDimmerControls(controls: HTMLElement, device: LutronDevice) {
        const commandType = this.levelCommandType(device);
        const row = document.createElement('div');
        row.className = 'control-row';

        const sliderWrap = document.createElement('div');
        sliderWrap.className = 'slider-wrap';

        this.slider = document.createElement('input');
        this.slider.type = 'range';
        this.slider.min = '0';
        this.slider.max = '100';
        this.slider.step = '1';
        this.slider.value = (device.Level ?? 0).toString();
        this.slider.addEventListener('input', () => {
            if (this.levelValue) {
                this.levelValue.textContent = `${this.slider.value}%`;
            }
        });
        this.slider.addEventListener('change', () => {
            const value = parseInt(this.slider.value, 10);
            this.applyLevel(device.Zone, value, commandType);
        });
        sliderWrap.appendChild(this.slider);

        this.levelValue = document.createElement('div');
        this.levelValue.className = 'level-value';
        this.levelValue.textContent = `${device.Level ?? 0}%`;
        sliderWrap.appendChild(this.levelValue);

        row.appendChild(sliderWrap);

        const buttonRow = document.createElement('div');
        buttonRow.className = 'button-row';

        const offButton = this.buildActionButton('Off', () => this.applyLevel(device.Zone, 0, commandType));
        const onButton = this.buildActionButton('On', () => this.applyLevel(device.Zone, 100, commandType));

        buttonRow.appendChild(offButton);
        buttonRow.appendChild(onButton);
        row.appendChild(buttonRow);

        controls.appendChild(row);
    }

    private buildSwitchControls(controls: HTMLElement, device: LutronDevice) {
        const buttonRow = document.createElement('div');
        buttonRow.className = 'button-row';
        const offButton = this.buildActionButton('Off', () => this.applyLevel(device.Zone, 0, 'GoToSwitchedLevel'));
        const onButton = this.buildActionButton('On', () => this.applyLevel(device.Zone, 100, 'GoToSwitchedLevel'));
        buttonRow.appendChild(offButton);
        buttonRow.appendChild(onButton);
        controls.appendChild(buttonRow);
    }

    private buildShadeControls(controls: HTMLElement, device: LutronDevice) {
        const levelRow = document.createElement('div');
        levelRow.className = 'shade-level-row';
        const levelLabel = document.createElement('div');
        levelLabel.className = 'shade-level-label';
        const levelValue = device.Level ?? 0;
        levelLabel.textContent = `Open ${levelValue}%`;
        const levelBar = document.createElement('div');
        levelBar.className = 'shade-level-bar';
        const levelFill = document.createElement('div');
        levelFill.className = 'shade-level-fill';
        levelFill.style.width = `${levelValue}%`;
        levelBar.appendChild(levelFill);
        levelRow.appendChild(levelLabel);
        levelRow.appendChild(levelBar);
        controls.appendChild(levelRow);

        const buttonRow = document.createElement('div');
        buttonRow.className = 'button-row';
        const upButton = this.buildActionButton('Raise', () => this.applyShadeCommand(device.Zone, 'Raise'));
        const stopButton = this.buildActionButton('Stop', () => this.applyShadeCommand(device.Zone, 'Stop'));
        const downButton = this.buildActionButton('Lower', () => this.applyShadeCommand(device.Zone, 'Lower'));
        buttonRow.appendChild(upButton);
        buttonRow.appendChild(stopButton);
        buttonRow.appendChild(downButton);
        controls.appendChild(buttonRow);
    }

    private levelCommandType(device: LutronDevice): string {
        if (device.DeviceType === 'WallSwitch') {
            return 'GoToSwitchedLevel';
        }
        if (device.DeviceType === 'WallDimmer') {
            return 'GoToDimmedLevel';
        }
        return 'GoToDimmedLevel';
    }

    private buildButtonControls(buttons: ButtonInfo[]) {
        const section = document.createElement('div');
        section.className = 'button-section';

        const title = document.createElement('div');
        title.className = 'button-section-title';
        title.textContent = 'Buttons';
        section.appendChild(title);

        const grid = document.createElement('div');
        grid.className = 'button-grid';
        buttons.forEach((button) => {
            const label = button.Name || `Button ${button.ButtonNumber}`;
            const action = this.buildActionButton(label, () => this.pressButton(button.ButtonNumber));
            grid.appendChild(action);
        });
        section.appendChild(grid);
        this.element.appendChild(section);
    }

    private buildActionButton(label: string, onClick: () => void): HTMLButtonElement {
        const button = document.createElement('button');
        button.className = 'action-button';
        button.textContent = label;
        button.addEventListener('click', () => onClick());
        return button;
    }

    private async pressButton(buttonNumber: number) {
        if (this.busy) {
            return;
        }
        this.setBusy(true);
        try {
            await pressAndRelease(buttonNumber);
            this.notify(`Pressed button ${buttonNumber}`, 'success');
            this.refresh();
        } catch (e) {
            this.notify('' + e, 'error');
        } finally {
            this.setBusy(false);
        }
    }

    private async applyLevel(zoneHref: string, level: number, commandType: string) {
        if (this.busy) {
            return;
        }
        this.setBusy(true);
        try {
            await setLevel(zoneHref, level, commandType);
            if (this.slider) {
                this.slider.value = level.toString();
            }
            if (this.levelValue) {
                this.levelValue.textContent = `${level}%`;
            }
            this.notify(`Set to ${level}%`, 'success');
            this.refresh();
        } catch (e) {
            this.notify('' + e, 'error');
        } finally {
            this.setBusy(false);
        }
    }

    private async applyShadeCommand(zoneHref: string, commandType: string) {
        if (this.busy) {
            return;
        }
        this.setBusy(true);
        try {
            await sendZoneCommand(zoneHref, commandType);
            this.notify(`${commandType} command sent`, 'success');
            this.refresh();
        } catch (e) {
            this.notify('' + e, 'error');
        } finally {
            this.setBusy(false);
        }
    }

    private setBusy(busy: boolean) {
        this.busy = busy;
        if (busy) {
            this.element.classList.add('is-busy');
        } else {
            this.element.classList.remove('is-busy');
        }
        this.element.querySelectorAll('button, input').forEach((el) => {
            if (el instanceof HTMLButtonElement || el instanceof HTMLInputElement) {
                el.disabled = busy;
            }
        });
    }
}

window.app = new App();
