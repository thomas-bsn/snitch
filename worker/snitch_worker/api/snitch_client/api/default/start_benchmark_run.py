from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.benchmark_created import BenchmarkCreated
from ...models.benchmark_run_input import BenchmarkRunInput
from ...models.error import Error
from typing import cast



def _get_kwargs(
    *,
    body: BenchmarkRunInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/benchmarks/runs",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> BenchmarkCreated | Error:
    if response.status_code == 201:
        response_201 = BenchmarkCreated.from_dict(response.json())



        return response_201

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[BenchmarkCreated | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: BenchmarkRunInput,
    origin: str,
    x_csrf_token: str,

) -> Response[BenchmarkCreated | Error]:
    """  Administrateurs uniquement. Lance des modèles sur les mêmes clips du corpus via des tâches à modèle
    demandé : aucun résultat de production, aucune validation ni le modèle actif ne sont modifiés.
    Empreinte et diarisation : modèles du catalogue de la même catégorie (benchmark_model_invalid sinon)
    ; la diarisation a une étape de tâche à part (benchmark.diarization), jamais une diarisation de
    production. Transcription : noms de modèles du service, une tâche benchmark.transcription par clip
    certifié et par piste. Un seul run actif (benchmark_run_active). Refus : benchmark_transition_open
    (empreinte), benchmark_dataset_stale, benchmark_stage_unsupported_by_dataset. Bout en bout (stage
    e2e) : des chaînes (diarisation + transcription), composées à partir des tâches des deux étapes,
    partagées entre chaînes qui utilisent le même modèle ; mesure par le cpWER.

    Args:
        origin (str):
        x_csrf_token (str):
        body (BenchmarkRunInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BenchmarkCreated | Error]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    body: BenchmarkRunInput,
    origin: str,
    x_csrf_token: str,

) -> BenchmarkCreated | Error | None:
    """  Administrateurs uniquement. Lance des modèles sur les mêmes clips du corpus via des tâches à modèle
    demandé : aucun résultat de production, aucune validation ni le modèle actif ne sont modifiés.
    Empreinte et diarisation : modèles du catalogue de la même catégorie (benchmark_model_invalid sinon)
    ; la diarisation a une étape de tâche à part (benchmark.diarization), jamais une diarisation de
    production. Transcription : noms de modèles du service, une tâche benchmark.transcription par clip
    certifié et par piste. Un seul run actif (benchmark_run_active). Refus : benchmark_transition_open
    (empreinte), benchmark_dataset_stale, benchmark_stage_unsupported_by_dataset. Bout en bout (stage
    e2e) : des chaînes (diarisation + transcription), composées à partir des tâches des deux étapes,
    partagées entre chaînes qui utilisent le même modèle ; mesure par le cpWER.

    Args:
        origin (str):
        x_csrf_token (str):
        body (BenchmarkRunInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BenchmarkCreated | Error
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: BenchmarkRunInput,
    origin: str,
    x_csrf_token: str,

) -> Response[BenchmarkCreated | Error]:
    """  Administrateurs uniquement. Lance des modèles sur les mêmes clips du corpus via des tâches à modèle
    demandé : aucun résultat de production, aucune validation ni le modèle actif ne sont modifiés.
    Empreinte et diarisation : modèles du catalogue de la même catégorie (benchmark_model_invalid sinon)
    ; la diarisation a une étape de tâche à part (benchmark.diarization), jamais une diarisation de
    production. Transcription : noms de modèles du service, une tâche benchmark.transcription par clip
    certifié et par piste. Un seul run actif (benchmark_run_active). Refus : benchmark_transition_open
    (empreinte), benchmark_dataset_stale, benchmark_stage_unsupported_by_dataset. Bout en bout (stage
    e2e) : des chaînes (diarisation + transcription), composées à partir des tâches des deux étapes,
    partagées entre chaînes qui utilisent le même modèle ; mesure par le cpWER.

    Args:
        origin (str):
        x_csrf_token (str):
        body (BenchmarkRunInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BenchmarkCreated | Error]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    body: BenchmarkRunInput,
    origin: str,
    x_csrf_token: str,

) -> BenchmarkCreated | Error | None:
    """  Administrateurs uniquement. Lance des modèles sur les mêmes clips du corpus via des tâches à modèle
    demandé : aucun résultat de production, aucune validation ni le modèle actif ne sont modifiés.
    Empreinte et diarisation : modèles du catalogue de la même catégorie (benchmark_model_invalid sinon)
    ; la diarisation a une étape de tâche à part (benchmark.diarization), jamais une diarisation de
    production. Transcription : noms de modèles du service, une tâche benchmark.transcription par clip
    certifié et par piste. Un seul run actif (benchmark_run_active). Refus : benchmark_transition_open
    (empreinte), benchmark_dataset_stale, benchmark_stage_unsupported_by_dataset. Bout en bout (stage
    e2e) : des chaînes (diarisation + transcription), composées à partir des tâches des deux étapes,
    partagées entre chaînes qui utilisent le même modèle ; mesure par le cpWER.

    Args:
        origin (str):
        x_csrf_token (str):
        body (BenchmarkRunInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BenchmarkCreated | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
