from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.benchmark_model_list import BenchmarkModelList
from ...models.error import Error
from ...models.list_benchmark_models_stage import ListBenchmarkModelsStage
from typing import cast



def _get_kwargs(
    *,
    stage: ListBenchmarkModelsStage,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_stage = stage.value
    params["stage"] = json_stage


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/benchmarks/models",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> BenchmarkModelList | Error:
    if response.status_code == 200:
        response_200 = BenchmarkModelList.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[BenchmarkModelList | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    stage: ListBenchmarkModelsStage,

) -> Response[BenchmarkModelList | Error]:
    """  Administrateurs uniquement. Modèles du catalogue comparables pour l'étape, avec le nombre de Workers
    où chacun est prêt (un modèle prêt nulle part laisserait le run en attente).

    Args:
        stage (ListBenchmarkModelsStage):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BenchmarkModelList | Error]
     """


    kwargs = _get_kwargs(
        stage=stage,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    stage: ListBenchmarkModelsStage,

) -> BenchmarkModelList | Error | None:
    """  Administrateurs uniquement. Modèles du catalogue comparables pour l'étape, avec le nombre de Workers
    où chacun est prêt (un modèle prêt nulle part laisserait le run en attente).

    Args:
        stage (ListBenchmarkModelsStage):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BenchmarkModelList | Error
     """


    return sync_detailed(
        client=client,
stage=stage,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    stage: ListBenchmarkModelsStage,

) -> Response[BenchmarkModelList | Error]:
    """  Administrateurs uniquement. Modèles du catalogue comparables pour l'étape, avec le nombre de Workers
    où chacun est prêt (un modèle prêt nulle part laisserait le run en attente).

    Args:
        stage (ListBenchmarkModelsStage):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BenchmarkModelList | Error]
     """


    kwargs = _get_kwargs(
        stage=stage,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    stage: ListBenchmarkModelsStage,

) -> BenchmarkModelList | Error | None:
    """  Administrateurs uniquement. Modèles du catalogue comparables pour l'étape, avec le nombre de Workers
    où chacun est prêt (un modèle prêt nulle part laisserait le run en attente).

    Args:
        stage (ListBenchmarkModelsStage):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BenchmarkModelList | Error
     """


    return (await asyncio_detailed(
        client=client,
stage=stage,

    )).parsed
