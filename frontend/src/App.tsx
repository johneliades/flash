import React, { useState, useEffect, useRef } from "react";
import { toast, ToastContainer } from "react-toastify"; // Import toast
import TorrentTable from "./components/TorrentTable";
import FileInput from "./components/FileInput";
import { TorrentStatus } from "./interface/Torrent";

function App() {
  const [torrents, setTorrents] = useState<TorrentStatus[]>([]);

  // Create a ref to store the latest torrents state
  const torrentsRef = useRef<TorrentStatus[]>(torrents);

  // Sync ref with state on each render
  useEffect(() => {
    torrentsRef.current = torrents;
  }, [torrents]);

  const startDownload = async (file: File) => {
    const formData = new FormData();
    formData.append("torrent", file);

    try {
      const response = await fetch("http://localhost:8080/start-download", {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        throw new Error("Failed to upload and start download");
      }

      await response.json();
    } catch (error) {
      console.error("Error:", error);
    }
  };

  const addTorrent = (file: File) => {
    const newTorrent: TorrentStatus = {
      name: file.name,
      progress: 0,
      downloadSpeed: 0,
      size: 0,
    };
    setTorrents((prevTorrents) => [...prevTorrents, newTorrent]);
    startDownload(file);
  };

  useEffect(() => {
    const intervalId = setInterval(async () => {
      try {
        for (let torrent of torrents) {
          const response = await fetch(
            `http://localhost:8080/download-progress?torrent_name=${torrent.name}`
          );

          if (response.ok) {
            const progressData: TorrentStatus = await response.json();
            // console.log(progressData);

            if (progressData.progress === 100) {
              setTorrents((prevTorrents) =>
                prevTorrents.filter((t) => t.name !== progressData.name)
              );

              // Show a toast notification when download is finished
              toast.success(`${progressData.name} downloaded!`);
              console.log(`${progressData.name} downloaded!`);

              continue; // Skip to the next iteration
            }

            setTorrents((prevTorrents) =>
              prevTorrents.map((t) =>
                t.name === progressData.name ? { ...t, ...progressData } : t
              )
            );
          } else {
            console.error("Failed to fetch progress:", response.statusText);
          }
        }
      } catch (error) {
        console.error("Error fetching progress:", error);
      }
    }, 2000);

    return () => clearInterval(intervalId);
  }, [torrents]);

  return (
    <div className="container mt-5 text-center">
      <h1 className="mb-4">Flash Torrent</h1>

      <TorrentTable torrents={torrents} />
      <FileInput onFileSelect={addTorrent} />

      {/* Add the ToastContainer here */}
      <ToastContainer />
    </div>
  );
}

export default App;