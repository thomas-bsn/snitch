from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.benchmark_activation_preview import BenchmarkActivationPreview
from ...models.error import Error
from ...models.preview_benchmark_activation_stage import PreviewBenchmarkActivationStage
from typing import cast



def _get_kwargs(
    *,
    stage: PreviewBenchmarkActivationStage,
    model: str,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_stage = stage.value
    params["stage"] = json_stage

    params["model"] = model


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/benchmarks/activation",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> BenchmarkActivationPreview | Error:
    if response.status_code == 200:
        response_200 = BenchmarkActivationPreview.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[BenchmarkActivationPreview | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    stage: PreviewBenchmarkActivationStage,
    model: str,

) -> Response[BenchmarkActivationPreview | Error]:
    """  Administrateurs uniquement. Aperçu du changement du modèle de production (transcription ou
    diarisation) : modèle remplacé, Workers prêts, clips qu'une réanalyse mettrait en jeu dont ceux qui
    portent un travail humain. L'empreinte passe par sa transition (benchmark_activation_unsupported).

    Args:
        stage (PreviewBenchmarkActivationStage):
        model (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BenchmarkActivationPreview | Error]
     """


    kwargs = _get_kwargs(
        stage=stage,
model=model,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    stage: PreviewBenchmarkActivationStage,
    model: str,

) -> BenchmarkActivationPreview | Error | None:
    """  Administrateurs uniquement. Aperçu du changement du modèle de production (transcription ou
    diarisation) : modèle remplacé, Workers prêts, clips qu'une réanalyse mettrait en jeu dont ceux qui
    portent un travail humain. L'empreinte passe par sa transition (benchmark_activation_unsupported).

    Args:
        stage (PreviewBenchmarkActivationStage):
        model (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BenchmarkActivationPreview | Error
     """


    return sync_detailed(
        client=client,
stage=stage,
model=model,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    stage: PreviewBenchmarkActivationStage,
    model: str,

) -> Response[BenchmarkActivationPreview | Error]:
    """  Administrateurs uniquement. Aperçu du changement du modèle de production (transcription ou
    diarisation) : modèle remplacé, Workers prêts, clips qu'une réanalyse mettrait en jeu dont ceux qui
    portent un travail humain. L'empreinte passe par sa transition (benchmark_activation_unsupported).

    Args:
        stage (PreviewBenchmarkActivationStage):
        model (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BenchmarkActivationPreview | Error]
     """


    kwargs = _get_kwargs(
        stage=stage,
model=model,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    stage: PreviewBenchmarkActivationStage,
    model: str,

) -> BenchmarkActivationPreview | Error | None:
    """  Administrateurs uniquement. Aperçu du changement du modèle de production (transcription ou
    diarisation) : modèle remplacé, Workers prêts, clips qu'une réanalyse mettrait en jeu dont ceux qui
    portent un travail humain. L'empreinte passe par sa transition (benchmark_activation_unsupported).

    Args:
        stage (PreviewBenchmarkActivationStage):
        model (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BenchmarkActivationPreview | Error
     """


    return (await asyncio_detailed(
        client=client,
stage=stage,
model=model,

    )).parsed
