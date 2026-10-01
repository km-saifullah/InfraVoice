import { useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { AiApi, InfrastructureApi } from '../lib/endpoints'
import { VoiceRecorderPanel } from '../components/voice/VoiceRecorderPanel'
import { SpecSummary } from '../components/specification/SpecSummary'
import { SpecJSONPanel } from '../components/specification/SpecJSONPanel'
import { Card } from '../components/ui/Card'
import { TextArea } from '../components/ui/TextField'
import { Button } from '../components/ui/Button'
import { Alert } from '../components/ui/Alert'

export function VoiceCommandPage() {
  const { project } = useOutletContext()
  const navigate = useNavigate()

  const [parsed, setParsed] = useState(null) // { transcript, needs_clarification, clarification, specification }
  const [specText, setSpecText] = useState('')
  const [specJsonError, setSpecJsonError] = useState(null)
  const [saveError, setSaveError] = useState(null)
  const [isSaving, setIsSaving] = useState(false)
  const [savedSpecId, setSavedSpecId] = useState(null)

  const [typedCommand, setTypedCommand] = useState('')
  const [isParsingText, setIsParsingText] = useState(false)
  const [textError, setTextError] = useState(null)

  function handleParsed(result) {
    setParsed(result)
    setSaveError(null)
    setSavedSpecId(null)
    setSpecJsonError(null)
    setSpecText(result.specification ? JSON.stringify(result.specification, null, 2) : '')
  }

  async function handleParseTyped() {
    setIsParsingText(true)
    setTextError(null)
    try {
      const result = await AiApi.parse(project.id, typedCommand)
      handleParsed(result)
      setTypedCommand('')
    } catch (error) {
      setTextError(error.message)
    } finally {
      setIsParsingText(false)
    }
  }

  async function handleSaveSpecification() {
    setSaveError(null)
    let specObject

    try {
      specObject = JSON.parse(specText)
    } catch {
      setSpecJsonError('This is not valid JSON.')
      return
    }

    setSpecJsonError(null)
    setIsSaving(true)

    try {
      const { specification } = await InfrastructureApi.create(project.id, specObject)
      setSavedSpecId(specification.id)
    } catch (error) {
      setSaveError(error.message)
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="mx-auto grid max-w-4xl gap-6">
      <Card
        title="Speak a command"
        subtitle={`e.g. "Create a t3 micro EC2 instance in a new private subnet"`}
      >
        <VoiceRecorderPanel projectId={project.id} onParsed={handleParsed} />

        <div className="mt-6 border-t border-ink-800 pt-5">
          <p className="mb-2 text-xs font-medium uppercase text-mist-500">Or type a command</p>
          <div className="flex gap-2">
            <TextArea
              className="flex-1"
              rows={2}
              value={typedCommand}
              onChange={(event) => setTypedCommand(event.target.value)}
              placeholder="Create an S3 bucket with versioning enabled"
            />
            <Button
              onClick={handleParseTyped}
              isLoading={isParsingText}
              disabled={!typedCommand.trim()}
            >
              Parse
            </Button>
          </div>
          {textError && (
            <Alert tone="error" className="mt-2">
              {textError}
            </Alert>
          )}
        </div>
      </Card>

      {parsed?.needs_clarification && (
        <Alert tone="warning">
          The AI needs more detail: <strong>{parsed.clarification}</strong> Try recording or typing
          again with that detail included.
        </Alert>
      )}

      {parsed?.specification && (
        <Card
          title="Review the generated specification"
          subtitle={parsed.transcript ? `Transcript: "${parsed.transcript}"` : undefined}
        >
          <SpecSummary spec={parsed.specification} />

          <div className="mt-5">
            <SpecJSONPanel value={specText} onChange={setSpecText} error={specJsonError} />
          </div>

          {saveError && (
            <Alert tone="error" className="mt-4">
              {saveError}
            </Alert>
          )}

          {savedSpecId ? (
            <Alert tone="info" className="mt-4">
              Specification saved. Head to{' '}
              <button
                className="underline"
                onClick={() => navigate(`/projects/${project.id}/terraform`, { state: { specId: savedSpecId } })}
              >
                Terraform runs
              </button>{' '}
              to plan and apply it.
            </Alert>
          ) : (
            <Button onClick={handleSaveSpecification} isLoading={isSaving} className="mt-4">
              Save specification
            </Button>
          )}
        </Card>
      )}
    </div>
  )
}
