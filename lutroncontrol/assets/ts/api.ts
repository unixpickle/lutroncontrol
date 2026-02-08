type LutronDeviceType = (
    "SmartBridge" | "WallDimmer" | "WallSwitch" | "Pico2Button" |
    "Pico3ButtonRaiseLower" | string
);

class RemoteError extends Error {
    constructor(msg: string) {
        super(msg)

        // Set the prototype explicitly for compatibility with older JS environments
        Object.setPrototypeOf(this, RemoteError.prototype);
    }
}

interface LutronDevice {
    FullyQualifiedName: NonNullable<string[]>;
    DeviceType: NonNullable<LutronDeviceType>;
    Level?: number;
    Zone?: string;
    Buttons?: ButtonInfo[];
}

interface ButtonInfo {
    Href: string;
    Name: string;
    ButtonNumber: number;
    ProgrammingModel?: ProgrammingModel;
}

interface ProgrammingModel {
    Href: string;
    ProgrammingModelType: string;
    Direction?: string;
    Preset?: Preset;
    PressPreset?: Preset;
    ReleasePreset?: Preset;
}

interface Preset {
    Href: string;
    DimmedLevelAssignments: DimmedLevelAssignment[];
    SwitchedLevelAssignments: SwitchedLevelAssignment[];
}

interface DimmedLevelAssignment {
    Href: string;
    FadeTime: string;
    DelayTime: string;
    Level: number;
}

interface SwitchedLevelAssignment {
    Href: string;
    DelayTime: string;
    SwitchedLevel: string;
}

interface SceneInfo {
    href: string;
    Name: string;
    IsProgrammed: boolean;
    ButtonNumber: number;
}

async function fetchDevices(): Promise<LutronDevice[]> {
    return fetchAPI<LutronDevice[]>('devices');
}

async function fetchScenes(): Promise<SceneInfo[]> {
    return fetchAPI<SceneInfo[]>('scenes');
}

async function fetchAPI<T>(url: string): Promise<T> {
    const obj = await (await fetch(url)).json();
    if (obj.hasOwnProperty("error")) {
        throw new RemoteError(obj["error"]);
    }
    return obj as T;
}

function deviceRoom(device: LutronDevice): string {
    if (device.FullyQualifiedName.length === 1) {
        return "Other";
    } else {
        return device.FullyQualifiedName[0];
    }
}

function deviceName(device: LutronDevice): string {
    if (device.FullyQualifiedName.length === 0) {
        return "Unnamed Device";
    }
    return device.FullyQualifiedName[device.FullyQualifiedName.length - 1];
}

function hrefId(href?: string): string | null {
    if (!href) {
        return null;
    }
    const parts = href.split('/').filter((part) => part.length > 0);
    if (parts.length === 0) {
        return null;
    }
    return parts[parts.length - 1];
}

async function setLevel(zoneHref: string, level: number, commandType: string): Promise<boolean> {
    const zoneId = hrefId(zoneHref);
    if (!zoneId) {
        throw new RemoteError("invalid zone reference");
    }
    const url = `command/set_level?type=${encodeURIComponent(commandType)}` +
        `&zone=${encodeURIComponent(zoneId)}` +
        `&level=${encodeURIComponent(level.toString())}`;
    return fetchAPI<boolean>(url);
}

async function sendZoneCommand(zoneHref: string, commandType: string): Promise<boolean> {
    const zoneId = hrefId(zoneHref);
    if (!zoneId) {
        throw new RemoteError("invalid zone reference");
    }
    const url = `command/set_level?type=${encodeURIComponent(commandType)}` +
        `&zone=${encodeURIComponent(zoneId)}`;
    return fetchAPI<boolean>(url);
}

async function pressAndRelease(buttonNumber: number): Promise<boolean> {
    const url = `command/press_and_release?button=${encodeURIComponent(buttonNumber.toString())}`;
    return fetchAPI<boolean>(url);
}

async function activateScene(sceneHref: string): Promise<boolean> {
    const sceneId = hrefId(sceneHref);
    if (!sceneId) {
        throw new RemoteError("invalid scene reference");
    }
    const url = `scene/activate?scene=${encodeURIComponent(sceneId)}`;
    return fetchAPI<boolean>(url);
}

async function allOff(): Promise<boolean> {
    return fetchAPI<boolean>('command/all_off');
}
