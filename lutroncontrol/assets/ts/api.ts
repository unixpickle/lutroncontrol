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
}

async function fetchDevices(): Promise<LutronDevice[]> {
    return fetchAPI<LutronDevice[]>('/devices');
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
