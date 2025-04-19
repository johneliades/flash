import React from "react";
import { TorrentStatus } from "../interface/Torrent"; 

interface TorrentRowProps {
  torrent: TorrentStatus;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec === 0) return "0 B/s";
  const k = 1024;
  const sizes = ["B/s", "KB/s", "MB/s", "GB/s", "TB/s"];
  const i = Math.floor(Math.log(bytesPerSec) / Math.log(k));
  return parseFloat((bytesPerSec / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function secondsToHuman(input: number) {
  const units = {
    year: 60 * 60 * 24 * 7 * 30 * 12,
    month: 60 * 60 * 24 * 7 * 30,
    week: 60 * 60 * 24 * 7,
    day: 60 * 60 * 24,
    hour: 60 * 60,
    minute: 60,
    second: 1,
  };

  let seconds = input;

  const years = Math.floor(seconds / units.year);
  seconds = Math.floor(seconds % units.year);

  const months = Math.floor(seconds / units.month);
  seconds = Math.floor(seconds % units.month);

  const weeks = Math.floor(seconds / units.week);
  seconds = Math.floor(seconds % units.week);

  const days = Math.floor(seconds / units.day);
  seconds = Math.floor(seconds % units.day);

  const hours = Math.floor(seconds / units.hour);
  seconds = Math.floor(seconds % units.hour);

  const minutes = Math.floor(seconds / units.minute);
  seconds = Math.floor(seconds % units.minute);

  const parts = [];

  if (years > 0) parts.push(`${years}y`);
  if (months > 0 || parts.length > 0) parts.push(`${months}mo`);
  if (weeks > 0 || parts.length > 0) parts.push(`${weeks}w`);
  if (days > 0 || parts.length > 0) parts.push(`${days}d`);
  if (hours > 0 || parts.length > 0) parts.push(`${hours}h`);
  if (minutes > 0 || parts.length > 0) parts.push(`${minutes}m`);
  parts.push(`${seconds}s`);

  return parts.join(" ");
}

const TorrentRow: React.FC<TorrentRowProps> = ({ torrent }) => {
  return (
    <tr>
      <td>{torrent.name.replace(/\.torrent$/, "")}</td>
      <td className="align-middle">
        <div
          className="progress"
          style={{
            height: "20px",
            position: "relative",
            border: "1.5px solid black", // Add a black border here
          }}
        >
          <div
            className="progress-bar"
            role="progressbar"
            style={{
              width: `${torrent.progress}%`,
              backgroundColor:
                `${torrent.progress}` != "100" ? "cyan" : "#66FF99",
            }}
            aria-valuenow={torrent.progress}
            aria-valuemin={0}
            aria-valuemax={100}
          ></div>
          <div
            style={{
              position: "absolute",
              top: "50%",
              left: "50%",
              transform: "translate(-50%, -50%)",
              color: "black",
            }}
          >
            {torrent.progress}%
          </div>
        </div>
      </td>
      <td className="text-center align-middle">
        {torrent.progress != 100 ? formatSpeed(torrent.downloadSpeed) : ""}
      </td>
      <td className="text-center align-middle">
        {torrent.progress != 100 ? formatBytes(torrent.size) : ""}
      </td>
      <td className="text-center align-middle">
        {torrent.downloadSpeed != 0
          ? secondsToHuman(torrent.size / torrent.downloadSpeed)
          : "∞"}
      </td>
    </tr>
  );
};

export default TorrentRow;
