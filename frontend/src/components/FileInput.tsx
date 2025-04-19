import React, { useRef } from "react";

interface FileInputProps {
  onFileSelect: (file: File) => void;
}

const FileInput: React.FC<FileInputProps> = ({ onFileSelect }) => {
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files ? event.target.files[0] : null;
    if (file) {
      onFileSelect(file);
    }
  };

  const handleAddTorrentClick = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  return (
    <>
      <button
        className="btn btn-primary btn-sm"
        onClick={handleAddTorrentClick}
        style={{ color: "cyan", backgroundColor: "black" }}
      >
        +
      </button>
      <input
        type="file"
        accept=".torrent"
        onChange={handleFileChange}
        ref={fileInputRef}
        style={{ display: "none" }}
      />
    </>
  );
};

export default FileInput;
