import React from "react";
import TorrentRow from "./TorrentRow";
import { Torrent } from '../interface/Torrent'; 

interface TorrentTableProps {
  torrents: Torrent[];
}

const TorrentTable: React.FC<TorrentTableProps> = ({ torrents }) => {
  return (
    <table className="table table-sm table-hover">
      <thead>
        <tr>
          <th style={{ width: "35%", textAlign: "center" }}>Torrent</th>
          <th style={{ width: "35%", textAlign: "center" }}>Progress</th>
          <th style={{ width: "10%", textAlign: "center" }}>Down</th>
          <th style={{ width: "10%", textAlign: "center" }}>Size</th>
          <th style={{ width: "10%", textAlign: "center" }}>ETA</th>
        </tr>
      </thead>
      <tbody>
        {torrents.map((torrent, idx) => (
          <TorrentRow key={idx} torrent={torrent} />
        ))}
      </tbody>
    </table>
  );
};

export default TorrentTable;
