import { useCallback, useEffect, useRef, useState } from "react";

const CANDIDATE_MIME_TYPES = [
  "audio/webm;codecs=opus",
  "audio/webm",
  "audio/ogg;codecs=opus",
  "audio/mp4",
];

function pickSupportedMimeType() {
  if (typeof MediaRecorder === "undefined") return null;
  return (
    CANDIDATE_MIME_TYPES.find((type) => MediaRecorder.isTypeSupported(type)) ||
    ""
  );
}

function extensionForMimeType(mimeType) {
  if (mimeType.includes("ogg")) return "ogg";
  if (mimeType.includes("mp4")) return "m4a";
  return "webm";
}

const LEVEL_SAMPLE_COUNT = 28;

export function useAudioRecorder() {
  const [status, setStatus] = useState("idle"); // idle | requesting | recording | stopped
  const [error, setError] = useState(null);
  const [levels, setLevels] = useState(() =>
    new Array(LEVEL_SAMPLE_COUNT).fill(0.05),
  );
  const [result, setResult] = useState(null); // { blob, url, mimeType, extension, durationMs }

  const mediaRecorderRef = useRef(null);
  const streamRef = useRef(null);
  const audioContextRef = useRef(null);
  const analyserRef = useRef(null);
  const animationFrameRef = useRef(null);
  const chunksRef = useRef([]);
  const startedAtRef = useRef(0);

  const stopAnalysis = useCallback(() => {
    if (animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }
    if (audioContextRef.current) {
      audioContextRef.current.close().catch(() => {});
      audioContextRef.current = null;
    }
    analyserRef.current = null;
  }, []);

  const releaseStream = useCallback(() => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
  }, []);

  const sampleLevels = useCallback(() => {
    const analyser = analyserRef.current;
    if (!analyser) return;

    const data = new Uint8Array(analyser.frequencyBinCount);
    analyser.getByteTimeDomainData(data);

    let sumSquares = 0;
    for (let i = 0; i < data.length; i += 1) {
      const normalized = (data[i] - 128) / 128;
      sumSquares += normalized * normalized;
    }
    const amplitude = Math.min(1, Math.sqrt(sumSquares / data.length) * 4);

    setLevels((previous) => [...previous.slice(1), Math.max(0.05, amplitude)]);
    animationFrameRef.current = requestAnimationFrame(sampleLevels);
  }, []);

  const start = useCallback(async () => {
    setError(null);
    setResult(null);

    if (!navigator.mediaDevices?.getUserMedia) {
      setError(
        "Voice recording needs HTTPS or localhost. The browser blocks microphone access on " +
          `plain http:// for any other address (you're on ${window.location.origin}).`,
      );
      return;
    }

    setStatus("requesting");

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;

      const mimeType = pickSupportedMimeType();
      const recorder = mimeType
        ? new MediaRecorder(stream, { mimeType })
        : new MediaRecorder(stream);

      chunksRef.current = [];
      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) chunksRef.current.push(event.data);
      };

      recorder.onstop = () => {
        const finalMimeType = recorder.mimeType || "audio/webm";
        const blob = new Blob(chunksRef.current, { type: finalMimeType });
        const url = URL.createObjectURL(blob);
        setResult({
          blob,
          url,
          mimeType: finalMimeType,
          extension: extensionForMimeType(finalMimeType),
          durationMs: Date.now() - startedAtRef.current,
        });
        setStatus("stopped");
        releaseStream();
        stopAnalysis();
      };

      const AudioContextClass =
        window.AudioContext || window.webkitAudioContext;
      const audioContext = new AudioContextClass();
      const source = audioContext.createMediaStreamSource(stream);
      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 512;
      source.connect(analyser);
      audioContextRef.current = audioContext;
      analyserRef.current = analyser;

      mediaRecorderRef.current = recorder;
      startedAtRef.current = Date.now();
      recorder.start();
      setStatus("recording");
      animationFrameRef.current = requestAnimationFrame(sampleLevels);
    } catch (caughtError) {
      setStatus("idle");
      setError(
        caughtError?.name === "NotAllowedError"
          ? "Microphone access was denied. Allow microphone access in your browser to record a voice command."
          : caughtError?.message || "Could not access the microphone.",
      );
      releaseStream();
    }
  }, [releaseStream, sampleLevels, stopAnalysis]);

  const stop = useCallback(() => {
    if (
      mediaRecorderRef.current &&
      mediaRecorderRef.current.state !== "inactive"
    ) {
      mediaRecorderRef.current.stop();
    }
  }, []);

  const reset = useCallback(() => {
    if (result?.url) URL.revokeObjectURL(result.url);
    setResult(null);
    setStatus("idle");
    setError(null);
    setLevels(new Array(LEVEL_SAMPLE_COUNT).fill(0.05));
  }, [result]);

  useEffect(() => {
    return () => {
      releaseStream();
      stopAnalysis();
      if (result?.url) URL.revokeObjectURL(result.url);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return { status, error, levels, result, start, stop, reset };
}
