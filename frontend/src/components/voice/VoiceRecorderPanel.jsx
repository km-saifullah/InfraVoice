import { useState } from "react";
import { useAudioRecorder } from "../../hooks/useAudioRecorder";
import { AiApi, CommandsApi, SpeechApi } from "../../lib/endpoints";
import { Button } from "../ui/Button";
import { Alert } from "../ui/Alert";
import { Waveform } from "./Waveform";

export function VoiceRecorderPanel({ projectId, onParsed }) {
  const { status, error, levels, result, start, stop, reset } =
    useAudioRecorder();
  const [preview, setPreview] = useState(null);
  const [isWorking, setIsWorking] = useState(false);
  const [workError, setWorkError] = useState(null);

  const filename = result ? `command.${result.extension}` : "command.webm";

  async function handlePreview() {
    setIsWorking(true);
    setWorkError(null);
    try {
      const transcription = await SpeechApi.transcribe(
        projectId,
        result.blob,
        filename,
      );
      setPreview(transcription.text);
    } catch (caughtError) {
      setWorkError(caughtError.message);
    } finally {
      setIsWorking(false);
    }
  }

  async function handleSend() {
    setIsWorking(true);
    setWorkError(null);
    try {
      const parsed = await AiApi.parseVoice(projectId, result.blob, filename);
      onParsed(parsed);
      CommandsApi.createVoice(projectId, result.blob, filename).catch(() => {});
      reset();
      setPreview(null);
    } catch (caughtError) {
      setWorkError(caughtError.message);
    } finally {
      setIsWorking(false);
    }
  }

  function handleReset() {
    reset();
    setPreview(null);
    setWorkError(null);
  }

  return (
    <div className="flex flex-col items-center gap-5 py-4">
      <Waveform
        levels={levels}
        active={status === "recording"}
        className="w-full max-w-md"
      />

      {status === "idle" && (
        <Button onClick={start} className="px-8 py-3 text-base">
          Start recording
        </Button>
      )}

      {status === "requesting" && (
        <Button isLoading disabled className="px-8 py-3 text-base">
          Requesting microphone
        </Button>
      )}

      {status === "recording" && (
        <Button variant="danger" onClick={stop} className="px-8 py-3 text-base">
          Stop recording
        </Button>
      )}

      {status === "stopped" && result && (
        <div className="flex w-full max-w-md flex-col items-center gap-4">
          <audio controls src={result.url} className="w-full" />

          {preview && (
            <div className="w-full rounded-md border border-ink-700 bg-ink-850 px-4 py-3 text-sm text-mist-200">
              <span className="mb-1 block text-xs font-medium uppercase text-mist-500">
                Transcript preview
              </span>
              {preview}
            </div>
          )}

          <div className="flex w-full flex-col gap-2 sm:flex-row">
            <Button variant="ghost" onClick={handleReset} disabled={isWorking}>
              Re-record
            </Button>
            <Button
              variant="secondary"
              onClick={handlePreview}
              isLoading={isWorking}
              className="sm:flex-1"
            >
              Preview transcript
            </Button>
            <Button
              onClick={handleSend}
              isLoading={isWorking}
              className="sm:flex-1"
            >
              Send as command
            </Button>
          </div>
        </div>
      )}

      {error && (
        <Alert tone="error" className="w-full max-w-md">
          {error}
        </Alert>
      )}
      {workError && (
        <Alert tone="error" className="w-full max-w-md">
          {workError}
        </Alert>
      )}
    </div>
  );
}
